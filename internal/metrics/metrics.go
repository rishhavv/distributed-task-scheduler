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

	// Performance and Throughput Metrics
	TaskThroughput = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "task_throughput_total",
		Help: "Number of tasks completed per time window",
	}, []string{"task_type", "time_window"})

	ResourceUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_utilization_ratio",
		Help: "Resource utilization ratio for different resource types",
	}, []string{"resource_type", "worker_id"}) // resource_type: cpu, memory, etc.

	WorkerLoadBalance = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "worker_load_balance_ratio",
		Help: "Current load ratio for each worker compared to average",
	}, []string{"worker_id"})

	TaskDistributionFairness = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "task_distribution_fairness_ratio",
		Help: "Ratio indicating fairness of task distribution across workers",
	}, []string{"worker_id", "task_priority"})

	ScalabilityMetrics = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scalability_metrics",
		Help: "Various scalability indicators like queue growth rate",
	}, []string{"metric_type"}) // metric_type: queue_growth_rate, worker_efficiency, etc.

	ResourceSaturation = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "resource_saturation_ratio",
		Help: "Saturation levels for different resource types",
	}, []string{"resource_type", "worker_id"})
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

	TaskAssignmentLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "task_assignment_latency_seconds",
		Help:    "Time taken to assign a task to a worker",
		Buckets: prometheus.ExponentialBuckets(0.1, 2.0, 10),
	}, []string{"worker_id"})
)
