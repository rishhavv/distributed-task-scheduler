package coordinator

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rishhavv/dts/internal/metrics"
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
	algorithm     string

	// Channels for internal communication
	taskCh     chan types.Task
	workerCh   chan *Worker
	shutdownCh chan struct{}
}

func NewCoordinator(logger *logrus.Logger, algorithm string) *Coordinator {
	c := &Coordinator{
		tasks:      make(map[string]types.Task),
		workers:    make(map[string]*Worker),
		taskQueue:  make([]string, 0),
		logger:     logger,
		taskCh:     make(chan types.Task, 10000),
		workerCh:   make(chan *Worker, 100),
		shutdownCh: make(chan struct{}),
		algorithm:  algorithm,
	}

	// Start the task distribution goroutine
	go c.distributeTasksLoop()
	// Start the worker health check goroutine
	go c.healthCheckLoop()

	return c
}

// SubmitTask adds a new task to the system
func (c *Coordinator) SubmitTask(taskReq types.TaskSubmitRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	numTasks := taskReq.Number
	if numTasks <= 0 {
		numTasks = 1 // Default to 1 task if not specified
	}

	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%d", len(c.tasks)+1)
		workNumber := rand.Intn(10) + 1

		task := types.Task{
			ID:         taskID,
			Name:       string(taskReq.TaskType),
			Status:     types.TaskStatusPending,
			CreatedAt:  time.Now(),
			Value:      rand.Intn(1000), // Random value for task
			WorkNumber: workNumber,
		}

		if _, exists := c.tasks[taskID]; exists {
			continue // Skip if task ID already exists
		}

		c.tasks[taskID] = task
		c.taskQueue = append(c.taskQueue, taskID)
		metrics.TaskQueueLength.Inc()

		c.logger.WithFields(logrus.Fields{
			"task_id": taskID,
			"type":    taskReq.TaskType,
		}).Info("Task submitted")

		// Notify task distribution loop
		select {
		case c.taskCh <- task:
		default:
			c.logger.Warn("Task channel full, distribution may be delayed")
		}
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
	metrics.WorkerRegistrations.Add(1)
	metrics.WorkersActive.Inc()

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
	fmt.Println("Processed all pending tasks", len(c.taskQueue), len(availableWorkers))
	// Simple round-robin task distribution
	for i := 0; i < len(availableWorkers) && len(c.taskQueue) > 0; i++ {
		taskID := c.taskQueue[0]
		task, exists := c.tasks[taskID]
		if !exists {
			// Remove invalid task from queue
			c.taskQueue = c.taskQueue[1:]
			continue
		}

		//Find suitable worker (can be enhanced with better selection strategy)
		// worker := c.selectWorker(availableWorkers, task)
		// if worker == nil {
		// 	break
		// }

		//Assign task to worker using AssignTaskToWorker function
		worker, err := c.AssignTaskToWorker(task, c.algorithm, availableWorkers) // Using round-robin as default algorithm
		if err != nil {
			c.logger.WithError(err).Error("Failed to assign task to worker")
			continue
		}

		// Update worker status
		if worker, exists := c.workers[worker.ID]; exists {
			worker.Status = WorkerStatusBusy
			worker.CurrentTaskID = task.ID
			worker.TaskCount++
		}
		task.WorkerID = worker.ID
		task.Status = types.TaskStatusAssigned
		task.AssignedAt = time.Now()
		c.tasks[task.ID] = task
		metrics.TaskAssignmentLatency.WithLabelValues(worker.ID).Observe(time.Since(task.CreatedAt).Seconds())

		// Remove task from queue
		c.taskQueue = c.taskQueue[1:]
	}

	totalWorkers := float64(len(availableWorkers))
	for _, worker := range availableWorkers {
		// Calculate load balance ratio
		workerLoadRatio := float64(worker.TaskCount) / float64(c.getTotalTasks())
		metrics.WorkerLoadBalance.WithLabelValues(worker.ID).Set(workerLoadRatio)

		// Calculate task distribution fairness
		fairnessRatio := float64(worker.TaskCount) / totalWorkers
		metrics.TaskDistributionFairness.WithLabelValues(worker.ID, "normal").Set(fairnessRatio)
	}

	// Add scalability metrics
	queueGrowthRate := float64(len(c.taskQueue)) / float64(c.getTotalProcessedTasks()+1)
	metrics.ScalabilityMetrics.WithLabelValues("queue_growth_rate").Set(queueGrowthRate)
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

// func (c *Coordinator) assignTaskToWorker(task types.Task, worker *Worker) {
// 	timer := prometheus.NewTimer(metrics.TaskSchedulingLatency)
// 	defer timer.ObserveDuration()

// 	waitTime := time.Since(task.CreatedAt)
// 	metrics.TaskQueueWaitTime.WithLabelValues(task.Type).Observe(waitTime.Seconds())
// 	metrics.TaskQueueLength.Dec()
// 	task.Status = types.TaskStatusAssigned
// 	task.WorkerID = worker.ID
// 	c.tasks[task.ID] = task

// 	worker.Status = WorkerStatusBusy
// 	worker.CurrentTaskID = task.ID
// 	worker.TaskCount++

// 	c.logger.WithFields(logrus.Fields{
// 		"task_id":   task.ID,
// 		"worker_id": worker.ID,
// 	}).Info("types.Task assigned to worker")
// }

// UpdateTaskStatus handles status updates from workers
func (c *Coordinator) UpdateTaskStatus(taskID string, status TaskStatus, err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldStatus := c.tasks[taskID].Status
	metrics.TaskStatusTransitions.WithLabelValues(string(oldStatus), string(status)).Inc()

	if status == TaskStatusComplete {
		completionTime := time.Since(c.tasks[taskID].CreatedAt)
		metrics.TaskCompletionTime.WithLabelValues(c.tasks[taskID].Type).Observe(completionTime.Seconds())
	}

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
		} else {
			c.logger.WithFields(logrus.Fields{
				"task_id":   task.ID,
				"worker_id": task.WorkerID,
			}).Error("Worker not found while updating task status")
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
		if now.Sub(worker.LastHeartbeat) > 30*time.Second {
			metrics.WorkerDeregistrations.WithLabelValues("heartbeat_timeout").Inc()
			metrics.TaskReassignments.WithLabelValues("worker_timeout").Inc()
			metrics.WorkersActive.Dec()
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
		fmt.Println("Task already assigned, returning same task: ", worker.CurrentTaskID)
		task := c.tasks[worker.CurrentTaskID]
		// if
		return &task, nil
	}

	// Get next task from queue
	if len(c.taskQueue) == 0 {
		return nil, nil // No tasks available
	}
	task := c.tasks[c.taskQueue[0]]
	return &task, nil

	// Pop next task from queue
	// taskID := c.taskQueue[0]
	// c.taskQueue = c.taskQueue[1:]

	// task := c.tasks[taskID]
	// task.Status = types.TaskStatusAssigned
	// task.WorkerID = workerID
	// task.AssignedAt = time.Now()

	// // Update task in map
	// c.tasks[taskID] = task

	// // Update worker
	// worker.CurrentTaskID = taskID
	// worker.TaskCount++
	// c.workers[workerID] = worker

	// c.logger.WithFields(logrus.Fields{
	// 	"task_id":   taskID,
	// 	"worker_id": workerID,
	// }).Info("types.Task assigned to worker")

	// return &task, nil
}

// AssignTaskToWorker assigns a task to a worker using the specified scheduling algorithm
func (c *Coordinator) AssignTaskToWorker(task types.Task, algorithm string, availableWorkers []*Worker) (*Worker, error) {

	var selectedWorkerID string

	switch algorithm {
	case "random":
		// Randomly select one of the available algorithms
		algos := []string{"round-robin", "fcfs", "least-loaded", "priority", "consistent-hash"}
		randomAlgo := algos[rand.Intn(len(algos))]
		c.logger.Info("Selected algorithm: ", randomAlgo)
		return c.AssignTaskToWorker(task, randomAlgo, availableWorkers)

	case "round-robin":
		// Simple round robin - pick next worker in sequence
		selectedWorkerID = availableWorkers[len(c.taskQueue)%len(availableWorkers)].ID
		break

	case "fcfs":
		// First available worker gets the task
		selectedWorkerID = availableWorkers[0].ID

	case "least-loaded":
		// Pick worker with lowest task count
		minTasks := availableWorkers[0].TaskCount
		selectedWorkerID = availableWorkers[0].ID
		for _, worker := range availableWorkers {
			if worker.TaskCount < minTasks {
				minTasks = worker.TaskCount
				selectedWorkerID = worker.ID
			}
		}

	case "priority":
		// Assign high priority tasks to workers with more capabilities
		maxCaps := len(availableWorkers[0].Capabilities)
		selectedWorkerID = availableWorkers[0].ID
		for _, worker := range availableWorkers {
			if len(worker.Capabilities) > maxCaps {
				maxCaps = len(worker.Capabilities)
				selectedWorkerID = worker.ID
			}
		}

	case "consistent-hash":
		// Simple consistent hashing based on task ID
		hash := 0
		for _, c := range task.ID {
			hash = 31*hash + int(c)
		}
		selectedWorkerID = availableWorkers[hash%len(availableWorkers)].ID

	case "weighted-rr":
		// Weight based on worker capabilities
		totalWeight := 0
		weights := make([]int, len(availableWorkers))
		for i, worker := range availableWorkers {
			weight := len(worker.Capabilities)
			weights[i] = weight
			totalWeight += weight
		}

		// Pick worker based on weighted distribution
		target := len(c.taskQueue) % totalWeight
		cumulative := 0
		for i, weight := range weights {
			cumulative += weight
			if target < cumulative {
				selectedWorkerID = availableWorkers[i].ID
				break
			}
		}

	default:
		return nil, fmt.Errorf("unknown scheduling algorithm: %s", algorithm)
	}
	fmt.Println("Selected worker: ", selectedWorkerID)

	// Update worker status
	worker := c.workers[selectedWorkerID]
	return worker, nil
	// worker.Status = WorkerStatusBusy
	// worker.CurrentTaskID = task.ID
	// worker.TaskCount++

	// // Update task
	// task.WorkerID = selectedWorkerID
	// task.Status = types.TaskStatusAssigned
	// task.AssignedAt = time.Now()

	// return selectedWorkerID, nil
}

func (c *Coordinator) getTotalTasks() int {
	total := 0
	for _, worker := range c.workers {
		total += worker.TaskCount
	}
	return total
}

func (c *Coordinator) getTotalProcessedTasks() int {
	total := 0
	for _, task := range c.tasks {
		if task.Status == types.TaskStatusCompleted {
			total++
		}
	}
	return total
}
