package worker

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rishhavv/dts/internal/worker"
)

func main() {
	worker := worker.NewWorker("worker-1", "http://localhost:8080")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := worker.Start(); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan

	// Cleanup
	if err := worker.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
