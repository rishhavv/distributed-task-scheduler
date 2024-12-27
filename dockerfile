# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build coordinator, worker and metrics binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o coordinator cmd/coordinator/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o worker cmd/worker/main.go  
RUN CGO_ENABLED=0 GOOS=linux go build -o metrics cmd/metrics/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/coordinator .
COPY --from=builder /app/worker .
COPY --from=builder /app/metrics .

# Expose ports
EXPOSE 8080 2112

# # Build the Docker image
# docker build -t dts-services .

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

# docker run -d --name worker dts-services ./worker --workers 3 --server "http://coordinator:8080" --capabilities "general"