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
				return
			}
		}
	}

	mc.registry.MustRegister(gauge)
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
			}

			// Scrape worker metrics (assuming workers are on ports 8081, 8082, etc.)
			workerPorts := []string{"8081", "8082"} // Add more ports as needed
			for _, port := range workerPorts {
				workerMetrics, err := collector.scrapeEndpoint(fmt.Sprintf("http://localhost:%s/metrics", port))
				if err != nil {
					logger.Error(err)
					continue
				}
				collector.updateRegistry(workerMetrics)
			}
		}
	}()

	logger.Info("Metrics server starting on port 2112")
	if err := http.ListenAndServe(":2112", nil); err != nil {
		logger.Fatal("Error starting metrics server: ", err)
	}
}
