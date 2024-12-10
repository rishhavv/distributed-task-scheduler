package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rishhavv/dts/internal/coordinator"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	algo := flag.String("algo", "random", "algo to use while assigning tasks")
	flag.Parse()

	coord := coordinator.NewCoordinator(logger, *algo)
	server := coordinator.NewHttpServer(coord, logger)

	r := mux.NewRouter()
	server.RegisterRoutes(r)

	logger.Info("Starting coordinator server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Fatal(err)
	}
}
