package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rishhavv/dts/internal/types"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	ID           string
	Capabilities []string
	ServerURL    string // Coordinator URL
	httpClient   *http.Client
	logger       *logrus.Logger

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
		shutdownCh: make(chan struct{}),
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

	if resp.StatusCode != http.StatusOK {
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
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/workers/%s/heartbeat", w.ServerURL, w.ID),
		nil)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
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

	// TODO: Implement actual task processing logic here

	// Update final task status
	if err := w.updateTaskStatus(ctx, task.ID, types.TaskStatusCompleted, nil); err != nil {
		w.logger.WithError(err).Error("Failed to update task status to complete")
	}

	w.mu.Lock()
	w.Status = types.WorkerStatusIdle
	w.currentTaskID = ""
	w.mu.Unlock()

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
