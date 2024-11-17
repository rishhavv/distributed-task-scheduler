package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	"bytes"
    "encoding/json"
    "fmt"
    "time"

	"github.com/rishhavv/dts/internal/types"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	ID         string
	Status     TaskStatus
	ServerURL  string // Coordinator URL
	httpClient *http.Client
	logger     *logrus.Logger
	tasks      map[string] // Currently assigned tasks
	mu         sync.RWMutex
	lastPing   time.Time
	firstRun   time.Time
	registered bool
}

// Registration request/response structures




func (w *Worker) Register() error {
	w.logger.WithField("worker_id", w.ID).Info("Attempting to register with coordinator")

	// Prepare registration request
	regRequest := types.RegistrationRequest{
		ID:     w.ID,
		Status: "available",
	}

	// Marshal the request
	jsonData, err := json.Marshal(regRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	// Create HTTP request
	registerURL := fmt.Sprintf("%s/workers", w.ServerURL)
	req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Set timeout for registration request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	// Send registration request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var regResponse RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResponse); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	// Verify registration success
	if !regResponse.Success {
		return fmt.Errorf("registration rejected by coordinator")
	}

	// Initialize worker state
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Status = types.TaskStatusPending
	w.LastPing = time.Now()
	w.tasks = make(map[string]types.Task)
	w.registered = true

	w.logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
		"status":    w.Status,
	}).Info("Successfully registered with coordinator")

	return nil
}

// Helper method for retrying registration
func (w *Worker) RegisterWithRetry(maxAttempts int, retryDelay time.Duration) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := w.Register()
		if err == nil {
			return nil
		}

		lastErr = err
		w.logger.WithFields(logrus.Fields{
			"attempt":      attempt,
			"error":        err,
			"max_attempts": maxAttempts,
		}).Warn("Registration attempt failed, retrying...")

		if attempt < maxAttempts {
			time.Sleep(retryDelay)
		}
	}

	return fmt.Errorf("failed to register after %d attempts: %w", maxAttempts, lastErr)
}

// Usage example in worker initialization:
func NewWorker(id string, serverURL string) *Worker {
	return &Worker{
		ID:        id,
		ServerURL: serverURL,
		Status:    "initializing",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logrus.New(),
		mu:     sync.RWMutex{},
	}
}

// Example usage with retry:
func (w *Worker) Start() error {
	// Try to register with retry
	err := w.RegisterWithRetry(3, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Continue with worker initialization...
	return nil
}

// StartHeartbeat initiates the heartbeat goroutine
func (w *Worker) StartHeartbeat(ctx context.Context) error {
    ticker := time.NewTicker(5 * time.Second) // Configurable interval
    w.logger.Info("Starting heartbeat mechanism")

    go func() {
        for {
            select {
            case <-ctx.Done():
                w.logger.Info("Stopping heartbeat mechanism")
                ticker.Stop()
                return
            case <-ticker.C:
                if err := w.sendHeartbeat(); err != nil {
                    w.logger.WithError(err).Error("Failed to send heartbeat")
                    // Implement exponential backoff or reconnection logic here
                }
            }
        }
    }()

    return nil
}

// sendHeartbeat sends a single heartbeat to the coordinator
func (w *Worker) sendHeartbeat() error {
    w.mu.RLock()
    currentTasks := make(map[string]string)
    for taskID, task := range w.tasks {
        currentTasks[taskID] = string(task.Status)
    }
    w.mu.RUnlock()

    heartbeat := types.HeartbeatRequest{
        WorkerID:  w.ID,
        Status:    w.Status,
        TaskCount: len(w.tasks),
        Tasks:     currentTasks,
    }

    jsonData, err := json.Marshal(heartbeat)
    if err != nil {
        return fmt.Errorf("failed to marshal heartbeat: %w", err)
    }

    // Create heartbeat request
    url := fmt.Sprintf("%s/workers/%s/heartbeat", w.ServerURL, w.ID)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to create heartbeat request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")

    // Set timeout for heartbeat request
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    req = req.WithContext(ctx)

    resp, err := w.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send heartbeat: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    w.mu.Lock()
    w.LastPing = time.Now()
    w.mu.Unlock()

    return nil
}

// Usage in worker Start method
func (w *Worker) Start() error {
    // First register the worker
    if err := w.Register(); err != nil {
        return fmt.Errorf("failed to register worker: %w", err)
    }

    // Create a context with cancellation for cleanup
    ctx, cancel := context.WithCancel(context.Background())
    w.cancel = cancel // Store cancel function for cleanup

    // Start heartbeat mechanism
    if err := w.StartHeartbeat(ctx); err != nil {
        return fmt.Errorf("failed to start heartbeat: %w", err)
    }

    w.logger.Info("Worker successfully started")
    return nil
}

// Cleanup method
func (w *Worker) Stop() error {
    if w.cancel != nil {
        w.cancel() // This will stop the heartbeat goroutine
    }
    w.logger.Info("Worker stopped")
    return nil
}