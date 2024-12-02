package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
)

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Basic HTML header with some simple styling
	fmt.Fprintf(w, `
		<html>
		<head>
			<title>DTS Metrics</title>
			<style>
				body { font-family: sans-serif; margin: 40px; }
				.metric { margin-bottom: 20px; }
				.metric-name { font-weight: bold; color: #2c3e50; }
				.metric-value { margin-left: 20px; color: #34495e; }
				.metric-type { color: #7f8c8d; font-size: 0.9em; }
			</style>
		</head>
		<body>
			<h1>DTS Metrics Dashboard</h1>
	`)

	// Fetch metrics from the /metrics endpoint
	resp, err := http.Get("http://localhost:2112/metrics")
	if err != nil {
		fmt.Fprintf(w, "<p>Error fetching metrics: %v</p>", err)
		return
	}
	defer resp.Body.Close()

	// Parse metrics
	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "<p>Error parsing metrics: %v</p>", err)
		return
	}

	// Display metrics
	for name, mf := range metrics {
		fmt.Fprintf(w, `<div class="metric">`)
		fmt.Fprintf(w, `<div class="metric-name">%s</div>`, name)
		fmt.Fprintf(w, `<div class="metric-type">Type: %s</div>`, mf.GetType().String())

		for _, m := range mf.GetMetric() {
			fmt.Fprintf(w, `<div class="metric-value">`)

			// Display labels if present
			if len(m.GetLabel()) > 0 {
				fmt.Fprintf(w, "{")
				for i, label := range m.GetLabel() {
					if i > 0 {
						fmt.Fprintf(w, ", ")
					}
					fmt.Fprintf(w, "%s=%s", label.GetName(), label.GetValue())
				}
				fmt.Fprintf(w, "} ")
			}

			// Display value based on metric type
			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				fmt.Fprintf(w, "Value: %.2f", m.GetCounter().GetValue())
			case dto.MetricType_GAUGE:
				fmt.Fprintf(w, "Value: %.2f", m.GetGauge().GetValue())
			case dto.MetricType_HISTOGRAM:
				fmt.Fprintf(w, "Count: %d, Sum: %.2f", m.GetHistogram().GetSampleCount(), m.GetHistogram().GetSampleSum())
			}

			fmt.Fprintf(w, "</div>")
		}
		fmt.Fprintf(w, "</div>")
	}

	fmt.Fprintf(w, "</body></html>")
}

type MetricsCollector struct {
	registry *prometheus.Registry
	logger   *logrus.Logger
	mu       sync.Mutex
}

func NewMetricsCollector(logger *logrus.Logger) *MetricsCollector {
	return &MetricsCollector{
		registry: prometheus.NewRegistry(),
		logger:   logger,
	}
}

func (mc *MetricsCollector) scrapeEndpoint(url string) ([]*dto.MetricFamily, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching metrics from %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from %s: %w", url, err)
	}

	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error parsing metrics from %s: %w", url, err)
	}

	result := make([]*dto.MetricFamily, 0, len(metrics))
	for _, mf := range metrics {
		result = append(result, mf)
	}
	return result, nil
}

func (mc *MetricsCollector) updateRegistry(metrics []*dto.MetricFamily) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, mf := range metrics {
		switch mf.GetType() {
		case dto.MetricType_COUNTER:
			mc.updateCounter(mf)
		case dto.MetricType_GAUGE:
			mc.updateGauge(mf)
		case dto.MetricType_HISTOGRAM:
			mc.updateHistogram(mf)
		}
	}
}

func (mc *MetricsCollector) updateCounter(mf *dto.MetricFamily) {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: mf.GetName(),
		Help: mf.GetHelp(),
	})

	if existing, err := mc.registry.Gather(); err == nil {
		for _, m := range existing {
			if m.GetName() == mf.GetName() {
				return
			}
		}
	}

	mc.registry.MustRegister(counter)
	counter.Add(mf.GetMetric()[0].GetCounter().GetValue())
}

func (mc *MetricsCollector) updateGauge(mf *dto.MetricFamily) {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: mf.GetName(),
		Help: mf.GetHelp(),
	})

	if existing, err := mc.registry.Gather(); err == nil {
		for _, m := range existing {
			if m.GetName() == mf.GetName() {
				// Find the existing gauge in the registry and update it
				check := mc.registry.Unregister(prometheus.NewGauge(prometheus.GaugeOpts{
					Name: mf.GetName(),
					Help: mf.GetHelp(),
				}))
				if check == false {
					mc.logger.Warn("Failed to unregister gauge:", mf.GetName())
				}
			}
		}
	}

	mc.registry.MustRegister(gauge)
	if mf.GetName() == "coordinator_worker_registrations_total" {
		fmt.Println("Gauge:", mf.GetName(), mf.GetMetric()[0].GetGauge().GetValue(), "\n\n")
	}
	gauge.Set(mf.GetMetric()[0].GetGauge().GetValue())
}

func (mc *MetricsCollector) updateHistogram(mf *dto.MetricFamily) {
	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    mf.GetName(),
		Help:    mf.GetHelp(),
		Buckets: prometheus.DefBuckets,
	})

	if existing, err := mc.registry.Gather(); err == nil {
		for _, m := range existing {
			if m.GetName() == mf.GetName() {
				return
			}
		}
	}

	mc.registry.MustRegister(histogram)
	for _, metric := range mf.GetMetric() {
		histogram.Observe(metric.GetHistogram().GetSampleSum())
	}
}

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	collector := NewMetricsCollector(logger)
	http.Handle("/metrics", promhttp.HandlerFor(collector.registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", metricsHandler)

	// Periodically fetch and update metrics
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			// Scrape coordinator metrics
			coordMetrics, err := collector.scrapeEndpoint("http://localhost:8080/metrics")
			if err != nil {
				logger.Error(err)
			} else {
				collector.updateRegistry(coordMetrics)
				logger.Info("Updated coordinator metrics")
			}

			// Scrape worker metrics (assuming workers are on ports 8081, 8082, etc.)
			// workerPorts := []string{"8081", "8082"} // Add more ports as needed
			// for _, port := range workerPorts {
			// 	workerMetrics, err := collector.scrapeEndpoint(fmt.Sprintf("http://localhost:%s/metrics", port))
			// 	if err != nil {
			// 		logger.Error(err)
			// 		continue
			// 	}
			// 	collector.updateRegistry(workerMetrics)
			// }
		}
	}()

	logger.Info("Metrics server starting on port 2112")
	if err := http.ListenAndServe(":2112", nil); err != nil {
		logger.Fatal("Error starting metrics server: ", err)
	}
}
