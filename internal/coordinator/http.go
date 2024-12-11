package coordinator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	r.HandleFunc("/tasks/next/{workerID}", s.handleGetNextTask).Methods("GET")
	r.HandleFunc("/tasks/{taskID}/status", s.handleUpdateTaskStatus).Methods("PUT")
	r.HandleFunc("/workers", s.handleRegisterWorker).Methods("POST")
	r.HandleFunc("/workers/{workerID}/heartbeat", s.handleWorkerHeartbeat).Methods("POST")
	r.HandleFunc("/", s.welcomeMessage).Methods("GET")
	r.Handle("/metrics", promhttp.Handler())

}

func (s *HttpServer) handleSubmitTask(w http.ResponseWriter, r *http.Request) {
	var taskReq types.TaskSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&taskReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.coordinator.SubmitTask(taskReq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode("success")
}

func (s *HttpServer) handleGetNextTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workerID := vars["workerID"]

	task, err := s.coordinator.GetNextTask(workerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if task == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	fmt.Println("Task fetched: ", task)

	json.NewEncoder(w).Encode(task)
}

func (s *HttpServer) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskID"]

	var update struct {
		Status types.TaskStatus `json:"status"`
		Error  string           `json:"error,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var taskErr error
	if update.Error != "" {
		taskErr = fmt.Errorf(update.Error)
	}

	if err := s.coordinator.UpdateTaskStatus(taskID, TaskStatus(update.Status), taskErr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpServer) welcomeMessage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to the Distributed Task Scheduler!"))
}

func (s *HttpServer) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workerID := vars["workerID"]

	var heartbeat types.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
		http.Error(w, "Invalid heartbeat data", http.StatusBadRequest)
		return
	}

	worker, err := s.coordinator.GetWorkerStatus(workerID)
	if err != nil {
		http.Error(w, "Worker not found", http.StatusNotFound)
		return
	}

	worker.LastHeartbeat = time.Now()
	worker.Status = WorkerStatus(heartbeat.Status)

	for taskID, status := range heartbeat.Tasks {
		if err := s.coordinator.UpdateTaskStatus(taskID, TaskStatus(status), nil); err != nil {
			s.logger.WithError(err).WithField("task_id", taskID).Error("Failed to update task status")
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpServer) handleRegisterWorker(w http.ResponseWriter, r *http.Request) {
	var req types.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid registration data", http.StatusBadRequest)
		return
	}

	worker := &Worker{
		ID:            req.ID,
		Status:        WorkerStatusIdle,
		LastHeartbeat: time.Now(),
	}

	if err := s.coordinator.RegisterWorker(worker); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := types.RegistrationResponse{
		WorkerID: worker.ID,
		Success:  true,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
