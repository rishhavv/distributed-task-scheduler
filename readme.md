# Distributed Task Scheduler

A scalable distributed task scheduling system written in Go that efficiently distributes workloads across multiple workers using various scheduling algorithms.

## Features

- Multiple scheduling algorithms supported:
  - Random
  - Round-robin 
  - First-come-first-served (FCFS)
  - Least-loaded
  - Consistent hashing
- Dynamic worker registration and health monitoring
- Configurable worker capabilities and task requirements
- Real-time metrics and monitoring
- Graceful shutdown handling
- Support for different workload patterns:
  - CPU intensive
  - I/O intensive 
  - Memory intensive
  - Mixed workloads
- Workload generation patterns:
  - Steady
  - Bursty
  - Ramp-up/down
  - Sinusoidal

## Components

### Coordinator (`cmd/coordinator/main.go`)

The coordinator is responsible for:
- Task queue management
- Worker registration and health checks
- Task distribution using configured scheduling algorithm
- Metrics collection and monitoring
- HTTP API endpoints for task submission and monitoring

### Worker (`cmd/worker/main.go`)

Workers are responsible for:
- Executing assigned tasks
- Reporting task status and completion
- Health check responses
- Resource utilization metrics

### Metrics (`cmd/metrics/main.go`)

Dedicated metrics component for:
- Collecting system-wide metrics
- Prometheus integration
- Performance monitoring
- Resource utilization tracking

## Deployment

### Docker Deployment

The system can be deployed using Docker. A multi-stage Dockerfile is provided for efficient builds.

#### Building the Image
# # Build the Docker image
<!-- # docker build -t dts-services .

# # Run Coordinator
# docker run -d \
#     -p 8080:8080 \
#     --name coordinator \
#     dts-services ./coordinator

# # Run Worker (with flags)
# docker run -d \
#     --name worker \
#     dts-services ./worker \
#     --workers 3 \
#     --server "http://coordinator:8080" \
#     --capabilities "general"

# # Run Metrics
# docker run -d \
#     -p 2112:2112 \
#     --name metrics \
#     dts-services ./metrics


# docker run -d \
#     --network host \
#     --name worker \
#     dts-services ./worker \
#     --workers 3 \
#     --server "http://host.docker.internal:8080" \
#     --capabilities "general"


# docker run -d -p 8080:8080 --name coordinator dts-services ./coordinator --workload=true

# docker run -d --name worker dts-services ./worker --workers 3 --server "http://coordinator:8080" --capabilities "general" -->