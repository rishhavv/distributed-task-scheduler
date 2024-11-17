// package coordinator

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/gorilla/mux"
// 	"github.com/rishhavv/dts/internal/types"
// 	"github.com/sirupsen/logrus"
// )

// type Coordinator struct {
// 	tasks     map[string]types.Task
// 	workers   map[string]*types.Worker
// 	taskQueue []string // Queue of task IDs
// 	mu        sync.RWMutex
// 	logger    *logrus.Logger
// }

// func NewCoordinator(logger *logrus.Logger) *Coordinator {
// 	return &Coordinator{
// 		tasks:     make(map[string]types.Task),
// 		workers:   make(map[string]*types.Worker),
// 		taskQueue: make([]string, 0),
// 		logger:    logger,
// 	}
// }

// func (c *Coordinator) SubmitTask(task types.Task) error {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	if task.ID == "" {
// 		task.ID = uuid.New().String()
// 	}
// 	task.CreatedAt = time.Now()
// 	task.Status = types.TaskStatusPending

// 	c.tasks[task.ID] = task
// 	c.taskQueue = append(c.taskQueue, task.ID)

// 	c.logger.WithFields(logrus.Fields{
// 		"taskID": task.ID,
// 		"type":   task.Type,
// 	}).Info("Task submitted")

// 	return nil
// }

// // In your coordinator package
// func (c *Coordinator) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	workerID := vars["workerID"]

// 	var heartbeat types.HeartbeatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
// 		http.Error(w, "Invalid heartbeat data", http.StatusBadRequest)
// 		return
// 	}

// 	c.mu.Lock()
// 	if worker, exists := c.workers[workerID]; exists {
// 		worker.LastPing = time.Now()
// 		worker.Status = heartbeat.Status
// 		// Update task statuses
// 		for taskID, status := range heartbeat.Tasks {
// 			if task, exists := c.tasks[taskID]; exists {
// 				task.Status = status
// 				c.tasks[taskID] = task
// 			}
// 		}
// 		c.workers[workerID] = worker
// 	} else {
// 		c.logger.WithField("worker_id", workerID).Warn("Heartbeat from unregistered worker")
// 		http.Error(w, "Worker not registered", http.StatusNotFound)
// 		c.mu.Unlock()
// 		return
// 	}
// 	c.mu.Unlock()

// 	w.WriteHeader(http.StatusOK)
// }

// // Add worker cleanup mechanism in coordinator
// func (c *Coordinator) StartWorkerCleanup(cleanupInterval time.Duration) {
// 	go func() {
// 		ticker := time.NewTicker(cleanupInterval)
// 		for range ticker.C {
// 			c.cleanupInactiveWorkers()
// 		}
// 	}()
// }

// func (c *Coordinator) cleanupInactiveWorkers() {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	threshold := time.Now().Add(-15 * time.Second) // Consider workers inactive after 15s
// 	for workerID, worker := range c.workers {
// 		if worker.LastPing.Before(threshold) {
// 			c.logger.WithField("worker_id", workerID).Info("Removing inactive worker")
// 			// Reassign tasks from this worker
// 			for taskID, task := range c.tasks {
// 				if task.WorkerID == workerID {
// 					task.Status = "pending"
// 					task.WorkerID = ""
// 					c.tasks[taskID] = task
// 					c.taskQueue = append(c.taskQueue, taskID)
// 				}
// 			}
// 			delete(c.workers, workerID)
// 		}
// 	}
// }

// func (c *Coordinator) RegisterWorker(workerID string) (bool, error) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	// Check if worker already exists
// 	if _, exists := c.workers[workerID]; exists {
// 		return false, fmt.Errorf("worker %s already registered", workerID)
// 	}

// 	// Create new worker entry
// 	worker := &types.Worker{
// 		ID:       workerID,
// 		Status:   types.TaskStatusPending,
// 		LastPing: time.Now(),
// 		FirstRun: time.Now(),
// 	}

// 	// Add worker to coordinator's worker map
// 	c.workers[workerID] = worker

// 	c.logger.WithField("worker_id", workerID).Info("New worker registered")
// 	return true, nil
// }

// // Add other methods for worker management and task 