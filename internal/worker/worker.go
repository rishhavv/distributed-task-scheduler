package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rishhavv/dts/internal/metrics"
	"github.com/rishhavv/dts/internal/types"
	"github.com/rishhavv/dts/tasks"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	ID                 string
	Capabilities       []string
	ServerURL          string // Coordinator URL
	httpClient         *http.Client
	logger             *logrus.Logger
	Tasks              map[string]types.TaskStatus
	lastTaskCompletion time.Time

	// Worker state
	Status        types.WorkerStatus
	currentTaskID string
	taskCount     int
	lastHeartbeat time.Time
	mu            sync.RWMutex

	// Control channels
	shutdownCh chan struct{}
}

type WorkerConfig struct {
	ID           string
	Capabilities []string
	ServerURL    string
	Logger       *logrus.Logger
}

type MetricsServer struct {
	worker *Worker
	logger *logrus.Logger
}

func NewMetricsServer(worker *Worker, logger *logrus.Logger) *MetricsServer {
	return &MetricsServer{
		worker: worker,
		logger: logger,
	}
}

func (s *MetricsServer) RegisterRoutes(r *mux.Router) {
	r.Handle("/metrics", promhttp.Handler())
}

func NewWorker(cfg WorkerConfig) *Worker {
	return &Worker{
		ID:           cfg.ID,
		Capabilities: cfg.Capabilities,
		ServerURL:    cfg.ServerURL,
		logger:       cfg.Logger,
		Status:       types.WorkerStatusIdle,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		shutdownCh:         make(chan struct{}),
		lastTaskCompletion: time.Now(),
	}
}

func (w *Worker) Register(ctx context.Context) error {
	worker := &Worker{
		ID:           w.ID,
		Status:       types.WorkerStatusIdle,
		Capabilities: w.Capabilities,
	}

	jsonData, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("failed to marshal worker: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/workers", w.ServerURL),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	w.logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
	}).Info("Successfully registered with coordinator")

	return nil
}

func (w *Worker) Start(ctx context.Context) error {
	if err := w.Register(ctx); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Start heartbeat routine
	go w.heartbeatLoop(ctx)

	// Start task polling routine
	go w.pollTasks(ctx)

	w.logger.Info("Worker started successfully")

	return nil
}

func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.shutdownCh:
			return
		case <-ticker.C:
			if err := w.sendHeartbeat(ctx); err != nil {
				w.logger.WithError(err).Error("Failed to send heartbeat")
			}
		}
	}
}

func (w *Worker) sendHeartbeat(ctx context.Context) error {
	startTime := time.Now()
	heartbeatReq := types.HeartbeatRequest{
		WorkerID:  w.ID,
		Status:    string(w.Status),
		TaskCount: w.taskCount,
		Tasks:     make(map[string]string),
	}
	for taskID, status := range w.Tasks {
		heartbeatReq.Tasks[taskID] = string(status)
		heartbeatReq.TaskCount++
	}

	reqBody, err := json.Marshal(heartbeatReq)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/workers/%s/heartbeat", w.ServerURL, w.ID),
		bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}
	fmt.Println("Request: ", fmt.Sprintf("%s/workers/%s/heartbeat", w.ServerURL, w.ID), req.Body)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	metrics.WorkerHeartbeatLatency.WithLabelValues(w.ID).Set(time.Since(startTime).Seconds())
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	w.mu.Lock()
	w.lastHeartbeat = time.Now()
	w.mu.Unlock()

	return nil
}

func (w *Worker) pollTasks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.shutdownCh:
			return
		case <-ticker.C:
			w.mu.RLock()
			if w.Status == types.WorkerStatusIdle {
				w.mu.RUnlock()
				if err := w.fetchAndProcessTask(ctx); err != nil {
					w.logger.WithError(err).Debug("No task available")
				}
			} else {
				w.mu.RUnlock()
			}
		}
	}
}

func (w *Worker) fetchAndProcessTask(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/tasks/next/%s", w.ServerURL, w.ID),
		nil)

	if err != nil {
		return fmt.Errorf("failed to create task request: %w", err)
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	if resp.StatusCode == http.StatusNoContent {
		return nil // No tasks available
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch task with status: %d", resp.StatusCode)
	}

	var task types.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return fmt.Errorf("failed to decode task: %w", err)
	}

	// Process task
	w.mu.Lock()
	w.Status = types.WorkerStatusBusy
	w.currentTaskID = task.ID
	w.taskCount++
	w.mu.Unlock()

	// Update task status to running
	if err := w.updateTaskStatus(ctx, task.ID, types.TaskStatusRunning, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update task status to running")
	}
	timer := prometheus.NewTimer(metrics.WorkerTaskDuration.WithLabelValues(w.ID, task.Type))
	defer timer.ObserveDuration()
	metrics.WorkerIdleTime.WithLabelValues(w.ID).Observe(time.Since(w.lastTaskCompletion).Seconds())
	// TODO: Implement actual task processing logic here
	_, err = tasks.RunWorkload(task.Type, w.ID, "")
	metrics.WorkerTasksProcessed.WithLabelValues(w.ID, task.Type).Inc()
	w.lastTaskCompletion = time.Now()

	// Update final task statusŒ
	if err := w.updateTaskStatus(ctx, task.ID, types.TaskStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update task status to complete")
	}

	w.mu.Lock()
	w.Status = types.WorkerStatusIdle
	w.currentTaskID = ""
	w.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics.ResourceUtilization.WithLabelValues("memory", w.ID).Set(float64(m.Alloc) / float64(m.Sys))
	metrics.ResourceUtilization.WithLabelValues("cpu", w.ID).Set(float64(runtime.NumGoroutine()))

	return nil
}

func (w *Worker) updateTaskStatus(ctx context.Context, taskID string, status types.TaskStatus, taskErr error) error {
	update := struct {
		Status types.TaskStatus `json:"status"`
		Error  string           `json:"error,omitempty"`
	}{
		Status: status,
	}
	if taskErr != nil {
		update.Error = taskErr.Error()
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/tasks/%s/status", w.ServerURL, taskID),
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create status update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status update failed with code: %d", resp.StatusCode)
	}

	return nil
}

func (w *Worker) Shutdown(ctx context.Context) error {
	close(w.shutdownCh)
	w.logger.Info("Worker shutdown complete")
	return nil
}

func (w *Worker) TaskSelector(taskName string, taskValue int) (int, error) {
	switch taskName {
	case "fibonacci":
		return w.countFibonacci(taskValue), nil
	case "factorial":
		return w.countFactorial(taskValue), nil
	case "wait":
		time.Sleep(time.Duration(taskValue) * time.Second)
		return taskValue, nil
	default:
		return 0, fmt.Errorf("unknown task: %s", taskName)
	}
}

func (w *Worker) countFactorial(n int) int {
	if n == 0 {
		return 1
	}
	return n * w.countFactorial(n-1)
}

func (w *Worker) countFibonacci(n int) int {
	if n == 0 || n == 1 {
		return n
	}
	return w.countFibonacci(n-1) + w.countFibonacci(n-2)
}
