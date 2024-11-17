// package worker

// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"time"

// 	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
// 	"github.com/sirupsen/logrus"
// )

// type TaskHandler func(context.Context, Task) error

// type Worker struct {
// 	ID           string
// 	Capabilities []string
// 	Coordinator  CoordinatorClient // Interface for communicating with coordinator
// 	TaskHandlers map[string]TaskHandler
// 	currentTask  *types.Task
// 	mu           sync.RWMutex
// 	logger       *logrus.Logger

// 	// Worker state
// 	status        types.WorkerStatus
// 	lastHeartbeat time.Time

// 	// Control channels
// 	shutdownCh chan struct{}
// 	taskCh     chan types.Task
// }

// type WorkerConfig struct {
// 	ID           string
// 	Capabilities []string
// 	Logger       *logrus.Logger
// 	Coordinator  CoordinatorClient
// }

// // CoordinatorClient interface defines how worker communicates with coordinator
// type CoordinatorClient interface {
// 	RegisterWorker(context.Context, *Worker) error
// 	UpdateTaskStatus(context.Context, string, types.TaskStatus, error) error
// 	SendHeartbeat(context.Context, string) error
// 	FetchTask(context.Context, string) (*types.Task, error)
// }

// func NewWorker(cfg WorkerConfig) *Worker {
// 	return &Worker{
// 		id:           cfg.ID,
// 		capabilities: cfg.Capabilities,
// 		coordinator:  cfg.Coordinator,
// 		taskHandlers: make(map[string]TaskHandler),
// 		logger:       cfg.Logger,
// 		status:       types.WorkerStatusIdle,
// 		shutdownCh:   make(chan struct{}),
// 		taskCh:       make(chan types.Task, 10),
// 	}
// }

// // RegisterTaskHandler registers a handler function for a specific task type
// func (w *Worker) RegisterTaskHandler(taskType string, handler TaskHandler) {
// 	w.mu.Lock()
// 	defer w.mu.Unlock()
// 	w.taskHandlers[taskType] = handler
// }

// // Start begins the worker's operation
// func (w *Worker) Start(ctx context.Context) error {
// 	// Register with coordinator
// 	err := w.coordinator.RegisterWorker(ctx, &Worker{
// 		ID:           w.ID,
// 		Capabilities: w.Capabilities,
// 		Status:       types.WorkerStatusIdle,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to register worker: %w", err)
// 	}

// 	// Start heartbeat routine
// 	go w.heartbeatLoop(ctx)

// 	// Start task polling routine
// 	go w.taskPollingLoop(ctx)

// 	w.logger.WithFields(logrus.Fields{
// 		"worker_id":    w.id,
// 		"capabilities": w.capabilities,
// 	}).Info("Worker started")

// 	return nil
// }

// // heartbeatLoop sends periodic heartbeats to coordinator
// func (w *Worker) heartbeatLoop(ctx context.Context) {
// 	ticker := time.NewTicker(15 * time.Second) // Heartbeat every 15 seconds
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-w.shutdownCh:
// 			return
// 		case <-ticker.C:
// 			err := w.coordinator.SendHeartbeat(ctx, w.id)
// 			if err != nil {
// 				w.logger.WithError(err).Error("Failed to send heartbeat")
// 			}
// 		}
// 	}
// }

// // taskPollingLoop continuously polls for new tasks when idle
// func (w *Worker) taskPollingLoop(ctx context.Context) {
// 	ticker := time.NewTicker(1 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case <-w.shutdownCh:
// 			return
// 		case <-ticker.C:
// 			w.mu.RLock()
// 			if w.status == WorkerStatusIdle {
// 				w.mu.RUnlock()
// 				if err := w.pollForTask(ctx); err != nil {
// 					w.logger.WithError(err).Debug("No task available")
// 				}
// 			} else {
// 				w.mu.RUnlock()
// 			}
// 		}
// 	}
// }

// // pollForTask attempts to fetch and process a new task
// func (w *Worker) pollForTask(ctx context.Context) error {
// 	task, err := w.coordinator.FetchTask(ctx, w.id)
// 	if err != nil {
// 		return err
// 	}
// 	if task == nil {
// 		return nil
// 	}

// 	return w.processTask(ctx, *task)
// }

// // processTask handles the execution of a task
// func (w *Worker) processTask(ctx context.Context, task Task) error {
// 	w.mu.Lock()
// 	w.status = WorkerStatusBusy
// 	w.currentTask = &task
// 	w.mu.Unlock()

// 	// Update task status to running
// 	err := w.coordinator.UpdateTaskStatus(ctx, task.ID, TaskStatusRunning, nil)
// 	if err != nil {
// 		w.logger.WithError(err).Error("Failed to update task status to running")
// 		return err
// 	}

// 	w.logger.WithFields(logrus.Fields{
// 		"task_id": task.ID,
// 		"type":    task.Type,
// 	}).Info("Processing task")

// 	// Get the appropriate handler for this task type
// 	w.mu.RLock()
// 	handler, exists := w.taskHandlers[task.Type]
// 	w.mu.RUnlock()

// 	if !exists {
// 		err := fmt.Errorf("no handler registered for task type: %s", task.Type)
// 		w.handleTaskCompletion(ctx, task, err)
// 		return err
// 	}

// 	// Execute the task with a timeout context
// 	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
// 	defer cancel()

// 	// Create error channel for task execution
// 	errCh := make(chan error, 1)

// 	// Execute task in goroutine
// 	go func() {
// 		errCh <- handler(taskCtx, task)
// 	}()

// 	// Wait for task completion or timeout
// 	var taskErr error
// 	select {
// 	case <-taskCtx.Done():
// 		taskErr = fmt.Errorf("task timeout: %w", taskCtx.Err())
// 	case err := <-errCh:
// 		taskErr = err
// 	}

// 	// Handle task completion
// 	w.handleTaskCompletion(ctx, task, taskErr)

// 	return taskErr
// }

// // handleTaskCompletion updates task status and worker state after task completion
// func (w *Worker) handleTaskCompletion(ctx context.Context, task Task, taskErr error) {
// 	status := TaskStatusComplete
// 	if taskErr != nil {
// 		status = TaskStatusFailed
// 	}

// 	// Update task status with coordinator
// 	err := w.coordinator.UpdateTaskStatus(ctx, task.ID, status, taskErr)
// 	if err != nil {
// 		w.logger.WithError(err).Error("Failed to update task completion status")
// 	}

// 	w.mu.Lock()
// 	w.status = WorkerStatusIdle
// 	w.currentTask = nil
// 	w.mu.Unlock()

// 	w.logger.WithFields(logrus.Fields{
// 		"task_id": task.ID,
// 		"status":  status,
// 		"error":   taskErr,
// 	}).Info("Task processing completed")
// }

// // GetStatus returns the current status of the worker
// func (w *Worker) GetStatus() WorkerStatus {
// 	w.mu.RLock()
// 	defer w.mu.RUnlock()
// 	return w.status
// }

// // GetCurrentTask returns the task currently being processed, if any
// func (w *Worker) GetCurrentTask() *Task {
// 	w.mu.RLock()
// 	defer w.mu.RUnlock()
// 	return w.currentTask
// }

// // Shutdown gracefully stops the worker
// func (w *Worker) Shutdown(ctx context.Context) error {
// 	w.logger.Info("Worker shutting down...")

// 	// Signal shutdown to all goroutines
// 	close(w.shutdownCh)

// 	// Wait for current task to complete if any
// 	w.mu.RLock()
// 	currentTask := w.currentTask
// 	w.mu.RUnlock()

// 	if currentTask != nil {
// 		w.logger.Info("Waiting for current task to complete...")
// 		// Wait with timeout for task completion
// 		timer := time.NewTimer(30 * time.Second)
// 		defer timer.Stop()

// 		for {
// 			select {
// 			case <-timer.C:
// 				return fmt.Errorf("shutdown timeout waiting for task completion")
// 			case <-time.After(100 * time.Millisecond):
// 				w.mu.RLock()
// 				if w.currentTask == nil {
// 					w.mu.RUnlock()
// 					w.logger.Info("Worker shutdown complete")
// 					return nil
// 				}
// 				w.mu.RUnlock()
// 			case <-ctx.Done():
// 				return ctx.Err()
// 			}
// 		}
// 	}

// 	w.logger.Info("Worker shutdown complete")
// 	return nil
// }

// // Example usage of the worker:
// func ExampleUsage() {
// 	// Create logger
// 	logger := logrus.New()

// 	// Create coordinator client (implementation needed)
// 	coordinator := &MyCoordinatorClient{}

// 	// Create worker config
// 	config := WorkerConfig{
// 		ID:           "worker-1",
// 		Capabilities: []string{"task-type-1", "task-type-2"},
// 		Logger:       logger,
// 		Coordinator:  coordinator,
// 	}

// 	// Create worker
// 	worker := NewWorker(config)

// 	// Register task handlers
// 	worker.RegisterTaskHandler("task-type-1", func(ctx context.Context, task Task) error {
// 		// Handle task-type-1
// 		return nil
// 	})

// 	worker.RegisterTaskHandler("task-type-2", func(ctx context.Context, task Task) error {
// 		// Handle task-type-2
// 		return nil
// 	})

// 	// Start worker
// 	ctx := context.Background()
// 	if err := worker.Start(ctx); err != nil {
// 		logger.WithError(err).Fatal("Failed to start worker")
// 	}

// 	// Shutdown worker gracefully when needed
// 	if err := worker.Shutdown(ctx); err != nil {
// 		logger.WithError(err).Error("Error during shutdown")
// 	}
// }
