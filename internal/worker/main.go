package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rishhavv/dts/internal/worker"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	// Configure and create new worker
	workerCfg := worker.WorkerConfig{
		ID:           "worker-1",
		Capabilities: []string{"general"},
		ServerURL:    "http://localhost:8080",
		Logger:       logger,
	}

	w := worker.NewWorker(workerCfg)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := w.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan

	// Cleanup
	if err := w.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
