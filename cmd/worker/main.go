package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/rishhavv/dts/internal/worker"
	"github.com/sirupsen/logrus"
)

func main() {
	// Add command-line flags
	numWorkers := flag.Int("workers", 1, "Number of workers to spawn")
	serverURL := flag.String("server", "http://localhost:8080", "Server URL")
	capabilities := flag.String("capabilities", "general", "Comma-separated list of capabilities")
	flag.Parse()

	logger := logrus.New()
	logger.SetOutput(os.Stdout)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	workers := make([]*worker.Worker, *numWorkers)

	// Spawn multiple workers
	for i := 0; i < *numWorkers; i++ {
		workerCfg := worker.WorkerConfig{
			ID:           fmt.Sprintf("worker-%d", i+1),
			Capabilities: []string{*capabilities},
			ServerURL:    *serverURL,
			Logger:       logger,
		}

		workers[i] = worker.NewWorker(workerCfg)
		wg.Add(1)

		// Start each worker in a goroutine
		go func(w *worker.Worker) {
			defer wg.Done()
			if err := w.Start(ctx); err != nil {
				log.Printf("Failed to start worker: %v", err)
			}
		}(workers[i])
	}

	// Start metrics servers for each worker
	for i, w := range workers {
		metricsServer := worker.NewMetricsServer(w, logger)
		r := mux.NewRouter()
		metricsServer.RegisterRoutes(r)

		port := 9200 + i
		go func(port int) {
			logger.Infof("Starting metrics server on :%d", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), r); err != nil {
				logger.Error(err)
			}
		}(port)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan

	// Cancel context to signal all workers to stop
	cancel()

	// Shutdown all workers
	for _, w := range workers {
		if err := w.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}

	// Wait for all workers to finish
	wg.Wait()
}
