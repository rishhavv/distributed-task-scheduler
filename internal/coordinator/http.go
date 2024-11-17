package coordinator

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rishhavv/dts/internal/types"
	"github.com/sirupsen/logrus"
)

type HttpServer struct {
	coordinator *Coordinator
	logger      *logrus.Logger
}

func NewHttpServer(coordinator *Coordinator, logger *logrus.Logger) *HttpServer {
	return &HttpServer{
		coordinator: coordinator,
		logger:      logger,
	}
}

func (s *HttpServer) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/tasks", s.handleSubmitTask).Methods("POST")
	// r.HandleFunc("/tasks", s.handleListTasks).Methods("GET")
	r.HandleFunc("/workers", s.handleRegisterWorker).Methods("POST")
	r.HandleFunc("/workers/{workerID}/heartbeat", s.handleWorkerHeartbeat).Methods("POST")
	r.HandleFunc("/", s.welcomeMessage).Methods("GET")
}

func (s *HttpServer) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var task types.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.coordinator.SubmitTask(task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (s *HttpServer) welcomeMessage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the Distributed Task Scheduler!"))
}

// In your coordinator package
func (c *Coordinator) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workerID := vars["workerID"]

	var heartbeat types.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
		http.Error(w, "Invalid heartbeat data", http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	if worker, exists := c.workers[workerID]; exists {
		worker.LastPing = time.Now()
		worker.Status = heartbeat.Status
		// Update task statuses
		for taskID, status := range heartbeat.Tasks {
			if task, exists := c.tasks[taskID]; exists {
				task.Status = status
				c.tasks[taskID] = task
			}
		}
		c.workers[workerID] = worker
	} else {
		c.logger.WithField("worker_id", workerID).Warn("Heartbeat from unregistered worker")
		http.Error(w, "Worker not registered", http.StatusNotFound)
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// Add worker cleanup mechanism in coordinator
func (c *Coordinator) StartWorkerCleanup(cleanupInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		for range ticker.C {
			c.cleanupInactiveWorkers()
		}
	}()
}

func (c *Coordinator) cleanupInactiveWorkers() {
	c.mu.Lock()
	defer c.mu.Unlock()

	threshold := time.Now().Add(-15 * time.Second) // Consider workers inactive after 15s
	for workerID, worker := range c.workers {
		if worker.LastPing.Before(threshold) {
			c.logger.WithField("worker_id", workerID).Info("Removing inactive worker")
			// Reassign tasks from this worker
			for taskID, task := range c.tasks {
				if task.WorkerID == workerID {
					task.Status = "pending"
					task.WorkerID = ""
					c.tasks[taskID] = task
					c.taskQueue = append(c.taskQueue, taskID)
				}
			}
			delete(c.workers, workerID)
		}
	}
}

func (s *HttpServer) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	registrationRequest := types.RegistrationRequest{}
	if err := json.NewDecoder(r.Body).Decode(&registrationRequest); err != nil {
		http.Error(w, "Invalid registration data", http.StatusBadRequest)
		return
	}
	success, err := s.coordinator.RegisterWorker(registrationRequest.ID)
	if err != nil {
		http.Error(w, "Failed to register worker", http.StatusInternalServerError)
		return
	}
	response := types.RegistrationResponse{
		WorkerID: registrationRequest.ID,
		Success:  success,
	}
	json.NewEncoder(w).Encode(response)
}

// Add other handlers simila´rly
