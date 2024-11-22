// Package metrics provides metrics for the coordinator and workers
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Worker Metrics
var (
	// Worker Status Metrics
	WorkersActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "workers_active_total",
		Help: "The total number of currently active workers",
	})

	WorkerTasksProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "worker_tasks_processed_total",
		Help: "The total number of processed tasks per worker",
	}, []string{"worker_id", "task_type"})

	WorkerTaskDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "worker_task_duration_seconds",
		Help:    "Time taken to process tasks by workers",
		Buckets: prometheus.ExponentialBuckets(0.1, 2.0, 10),
	}, []string{"worker_id", "task_type"})

	WorkerErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "worker_errors_total",
		Help: "The total number of errors encountered by workers",
	}, []string{"worker_id", "error_type"})

	WorkerHeartbeatLatency = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "worker_heartbeat_latency_seconds",
		Help: "Time since last heartbeat for each worker",
	}, []string{"worker_id"})

	WorkerIdleTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "worker_idle_time_seconds",
		Help:    "Time workers spend in idle state",
		Buckets: prometheus.LinearBuckets(0, 30, 10),
	}, []string{"worker_id"})
)

// Coordinator Metrics
var (
	// Task Queue Metrics
	TaskQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coordinator_task_queue_length",
		Help: "Current length of the task queue",
	})

	TaskQueueWaitTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "coordinator_task_queue_wait_time_seconds",
		Help:    "Time tasks spend in queue before being assigned",
		Buckets: prometheus.ExponentialBuckets(0.1, 2.0, 10),
	}, []string{"task_type"})

	// Scheduling Metrics
	TaskSchedulingLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "coordinator_task_scheduling_latency_seconds",
		Help:    "Time taken to schedule tasks to workers",
		Buckets: prometheus.LinearBuckets(0, 0.1, 10),
	})

	TaskReassignments = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coordinator_task_reassignments_total",
		Help: "Number of times tasks were reassigned due to worker failures",
	}, []string{"reason"})

	// Worker Management Metrics
	WorkerRegistrations = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coordinator_worker_registrations_total",
		Help: "Total number of worker registrations",
	})

	WorkerDeregistrations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "coordinator_worker_deregistrations_total",
		Help: "Total number of worker deregistrations",
	}, []string{"reason"})

	// Task Completion Metrics
	TaskCompletionTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "task_completion_time_seconds",
		Help:    "Total time from task submission to completion",
		Buckets: prometheus.ExponentialBuckets(0.1, 2.0, 10),
	}, []string{"task_type"})

	TaskStatusTransitions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "task_status_transitions_total",
		Help: "Number of task status transitions",
	}, []string{"from_status", "to_status"})
)
