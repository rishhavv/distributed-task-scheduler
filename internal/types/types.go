package types

import (
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type TaskStatus string
type WorkerStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusAssigned  TaskStatus = "assigned"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	Data        []byte     `json:"data"`
	WorkerID    string     `json:"worker_id,omitempty"`
	Error       string     `json:"error,omitempty"`
	Payload     []byte     `json:"payload,omitempty"`
	AssignedAt  time.Time  `json:"assigned_at,omitempty"`
	CompletedAt time.Time  `json:"completed_at,omitempty"`
}

type Worker struct {
	ID         string
	Status     TaskStatus
	ServerURL  string // Coordinator URL
	httpClient *http.Client
	logger     *logrus.Logger
	Tasks      map[string]TaskStatus // Currently assigned tasks
	mu         sync.RWMutex
	LastPing   time.Time
	FirstRun   time.Time
}

type RegistrationRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type RegistrationResponse struct {
	WorkerID string `json:"worker_id"`
	Success  bool   `json:"success"`
}

type HeartbeatRequest struct {
	WorkerID  string            `json:"worker_id"`
	Status    string            `json:"status"`
	TaskCount int               `json:"task_count"`
	Tasks     map[string]string `json:"tasks"` // taskID -> taskStatus
}

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusOffline WorkerStatus = "offline"
)
