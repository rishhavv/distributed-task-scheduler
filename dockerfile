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

CMD ["./coordinator"]   
