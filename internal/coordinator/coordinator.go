package coordinator

import (
	"fmt"
	"sync"
	"time"

	"github.com/rishhavv/dts/internal/types"
	"github.com/sirupsen/logrus"
)

type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusAssigned TaskStatus = "assigned"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "completed"
	TaskStatusFailed   TaskStatus = "failed"
)

type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusOffline WorkerStatus = "offline"
)

type Worker struct {
	ID            string       `json:"id"`
	Status        WorkerStatus `json:"status"`
	Capabilities  []string     `json:"capabilities"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	CurrentTaskID string       `json:"current_task_id"`
	TaskCount     int          `json:"task_count"`
}

type Coordinator struct {
	tasks         map[string]types.Task
	workers       map[string]*Worker
	taskQueue     []string // Queue of task IDs
	assignedQueue []string // Queue of assigned task IDs
	mu            sync.RWMutex
	logger        *logrus.Logger

	// Channels for internal communication
	taskCh     chan types.Task
	workerCh   chan *Worker
	shutdownCh chan struct{}
}

func NewCoordinator(logger *logrus.Logger) *Coordinator {
	c := &Coordinator{
		tasks:      make(map[string]types.Task),
		workers:    make(map[string]*Worker),
		taskQueue:  make([]string, 0),
		logger:     logger,
		taskCh:     make(chan types.Task, 100),
		workerCh:   make(chan *Worker, 10),
		shutdownCh: make(chan struct{}),
	}

	// Start the task distribution goroutine
	go c.distributeTasksLoop()
	// Start the worker health check goroutine
	go c.healthCheckLoop()

	return c
}

// SubmitTask adds a new task to the system
func (c *Coordinator) SubmitTask(task types.Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	task.Status = types.TaskStatusPending
	task.CreatedAt = time.Now()

	c.tasks[task.ID] = task
	c.taskQueue = append(c.taskQueue, task.ID)

	c.logger.WithFields(logrus.Fields{
		"task_id": task.ID,
		"type":    task.Type,
	}).Info("types.Task submitted")

	// Notify task distribution loop
	select {
	case c.taskCh <- task:
	default:
		c.logger.Warn("types.Task channel full, distribution may be delayed")
	}

	return nil
}

// RegisterWorker adds a new worker to the system
func (c *Coordinator) RegisterWorker(worker *Worker) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.workers[worker.ID]; exists {
		return fmt.Errorf("worker with ID %s already exists", worker.ID)
	}

	worker.Status = WorkerStatusIdle
	worker.LastHeartbeat = time.Now()
	c.workers[worker.ID] = worker

	c.logger.WithFields(logrus.Fields{
		"worker_id":    worker.ID,
		"capabilities": worker.Capabilities,
	}).Info("Worker registered")

	// Notify worker registration
	select {
	case c.workerCh <- worker:
	default:
		c.logger.Warn("Worker channel full")
	}

	return nil
}

// distributeTasksLoop continuously processes the task queue
func (c *Coordinator) distributeTasksLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.shutdownCh:
			return
		case <-ticker.C:
			c.processPendingTasks()
		}
	}
}

func (c *Coordinator) processPendingTasks() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.taskQueue) == 0 {
		return
	}

	// Find available workers
	availableWorkers := make([]*Worker, 0)
	for _, worker := range c.workers {
		if worker.Status == WorkerStatusIdle {
			availableWorkers = append(availableWorkers, worker)
		}
	}

	if len(availableWorkers) == 0 {
		return
	}

	// Simple round-robin task distribution
	for i := 0; i < len(c.taskQueue); i++ {
		taskID := c.taskQueue[0]
		task, exists := c.tasks[taskID]
		if !exists {
			// Remove invalid task from queue
			c.taskQueue = c.taskQueue[1:]
			continue
		}

		// Find suitable worker (can be enhanced with better selection strategy)
		worker := c.selectWorker(availableWorkers, task)
		if worker == nil {
			break
		}

		// Assign task to worker
		c.assignTaskToWorker(task, worker)

		// Remove task from queue
		c.taskQueue = c.taskQueue[1:]
	}
}

func (c *Coordinator) selectWorker(workers []*Worker, task types.Task) *Worker {
	// Simple selection strategy - choose worker with least tasks
	var selectedWorker *Worker
	minTasks := int(^uint(0) >> 1) // Max int

	for _, worker := range workers {
		if worker.Status != WorkerStatusIdle {
			continue
		}

		// Check if worker has required capabilities
		if !c.hasRequiredCapabilities(worker, task) {
			continue
		}

		if worker.TaskCount < minTasks {
			minTasks = worker.TaskCount
			selectedWorker = worker
		}
	}

	return selectedWorker
}

func (c *Coordinator) hasRequiredCapabilities(worker *Worker, task types.Task) bool {
	// Implement capability matching logic here
	return true // Simplified for this example
}

func (c *Coordinator) assignTaskToWorker(task types.Task, worker *Worker) {
	task.Status = types.TaskStatusAssigned
	task.WorkerID = worker.ID
	c.tasks[task.ID] = task

	worker.Status = WorkerStatusBusy
	worker.CurrentTaskID = task.ID
	worker.TaskCount++

	c.logger.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"worker_id": worker.ID,
	}).Info("types.Task assigned to worker")
}

// UpdateTaskStatus handles status updates from workers
func (c *Coordinator) UpdateTaskStatus(taskID string, status TaskStatus, err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	task, exists := c.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Status = types.TaskStatus(status)
	if err != nil {
		task.Error = err.Error()
	}
	if status == TaskStatusComplete || status == TaskStatusFailed {
		task.CompletedAt = time.Now()
		if worker, exists := c.workers[task.WorkerID]; exists {
			worker.Status = WorkerStatusIdle
			worker.CurrentTaskID = ""
		}
	}

	c.tasks[taskID] = task

	c.logger.WithFields(logrus.Fields{
		"task_id": taskID,
		"status":  status,
		"error":   err,
	}).Info("types.Task status updated")

	return nil
}

// HandleWorkerHeartbeat updates worker's last heartbeat time
func (c *Coordinator) HandleWorkerHeartbeat(workerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	worker, exists := c.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %s not found", workerID)
	}

	worker.LastHeartbeat = time.Now()
	return nil
}

// healthCheckLoop periodically checks worker health
func (c *Coordinator) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.shutdownCh:
			return
		case <-ticker.C:
			c.checkWorkersHealth()
		}
	}
}

func (c *Coordinator) checkWorkersHealth() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, worker := range c.workers {
		// If no heartbeat received in last minute, mark worker as offline
		if now.Sub(worker.LastHeartbeat) > 10*time.Second {
			worker.Status = WorkerStatusOffline

			// Reassign any tasks from this worker
			if worker.CurrentTaskID != "" {
				if task, exists := c.tasks[worker.CurrentTaskID]; exists {
					task.Status = types.TaskStatusPending
					task.WorkerID = ""
					c.tasks[task.ID] = task
					c.taskQueue = append(c.taskQueue, task.ID)
				}
			}

			// Optionally remove the worker
			delete(c.workers, id)

			c.logger.WithFields(logrus.Fields{
				"worker_id": id,
			}).Warn("Worker marked as offline due to missing heartbeat")
		}
	}
}

// Shutdown gracefully stops the coordinator
func (c *Coordinator) Shutdown() {
	close(c.shutdownCh)
}

// GetTaskStatus returns the current status of a task
func (c *Coordinator) GetTaskStatus(taskID string) (types.Task, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	task, exists := c.tasks[taskID]
	if !exists {
		return types.Task{}, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// GetWorkerStatus returns the current status of a worker
func (c *Coordinator) GetWorkerStatus(workerID string) (*Worker, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	worker, exists := c.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker %s not found", workerID)
	}

	return worker, nil
}

// GetNextTask assigns and returns the next available task for a worker
func (c *Coordinator) GetNextTask(workerID string) (*types.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify worker exists and is active
	worker, exists := c.workers[workerID]
	if !exists {
		return nil, fmt.Errorf("worker %s not found", workerID)
	}

	// Check if worker already has a task
	// Return the same task if it is already assigned
	if worker.CurrentTaskID != "" {
		task := c.tasks[worker.CurrentTaskID]
		// if
		return &task, nil
	}

	// Get next task from queue
	if len(c.taskQueue) == 0 {
		return nil, nil // No tasks available
	}

	// Pop next task from queue
	taskID := c.taskQueue[0]
	c.taskQueue = c.taskQueue[1:]

	task := c.tasks[taskID]
	task.Status = types.TaskStatusAssigned
	task.WorkerID = workerID
	task.AssignedAt = time.Now()

	// Update task in map
	c.tasks[taskID] = task

	// Update worker
	worker.CurrentTaskID = taskID
	worker.TaskCount++
	c.workers[workerID] = worker

	c.logger.WithFields(logrus.Fields{
		"task_id":   taskID,
		"worker_id": workerID,
	}).Info("types.Task assigned to worker")

	return &task, nil
}
