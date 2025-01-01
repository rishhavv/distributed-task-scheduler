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
	"google.golang.org/protobuf/proto"
)

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Modern HTML template with enhanced styling
	fmt.Fprintf(w, `
		<html>
		<head>
			<title>DTS Metrics Dashboard</title>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
			<style>
				:root {
					--primary-color: #2563eb;
					--secondary-color: #1e40af;
					--text-primary: #1f2937;
					--text-secondary: #4b5563;
					--bg-primary: #ffffff;
					--bg-secondary: #f3f4f6;
					--border-color: #e5e7eb;
				}

				* {
					margin: 0;
					padding: 0;
					box-sizing: border-box;
				}

				body {
					font-family: 'Inter', sans-serif;
					line-height: 1.5;
					color: var(--text-primary);
					background-color: var(--bg-secondary);
					padding: 2rem;
				}

				.container {
					max-width: 1200px;
					margin: 0 auto;
				}

				.header {
					background-color: var(--bg-primary);
					padding: 1.5rem 2rem;
					border-radius: 8px;
					box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
					margin-bottom: 2rem;
				}

				h1 {
					color: var(--primary-color);
					font-size: 1.875rem;
					font-weight: 600;
				}

				.metrics-grid {
					display: grid;
					grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
					gap: 1.5rem;
				}

				.metric {
					background-color: var(--bg-primary);
					padding: 1.5rem;
					border-radius: 8px;
					box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
					transition: transform 0.2s ease;
				}

				.metric:hover {
					transform: translateY(-2px);
				}

				.metric-name {
					font-size: 1.125rem;
					font-weight: 600;
					color: var(--primary-color);
					margin-bottom: 0.5rem;
					border-bottom: 2px solid var(--border-color);
					padding-bottom: 0.5rem;
				}

				.metric-type {
					display: inline-block;
					font-size: 0.875rem;
					color: var(--text-secondary);
					background-color: var(--bg-secondary);
					padding: 0.25rem 0.75rem;
					border-radius: 9999px;
					margin-bottom: 0.75rem;
				}

				.metric-help {
					color: var(--text-secondary);
					font-size: 0.875rem;
					margin-bottom: 1rem;
				}

				.metric-value {
					background-color: var(--bg-secondary);
					padding: 0.75rem;
					border-radius: 6px;
					margin-bottom: 0.5rem;
					font-family: monospace;
					font-size: 0.875rem;
				}

				.metric-labels {
					color: var(--secondary-color);
					font-weight: 500;
				}

				.error-message {
					background-color: #fee2e2;
					color: #dc2626;
					padding: 1rem;
					border-radius: 6px;
					margin-bottom: 1rem;
				}

				@media (max-width: 768px) {
					body {
						padding: 1rem;
					}
					
					.metrics-grid {
						grid-template-columns: 1fr;
					}
				}
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>DTS Metrics Dashboard</h1>
				</div>
				<div class="metrics-grid">
	`)

	// Fetch metrics from the /metrics endpoint
	resp, err := http.Get("http://localhost:2112/metrics")
	if err != nil {
		fmt.Fprintf(w, `<div class="error-message">Error fetching metrics: %v</div>`, err)
		return
	}
	defer resp.Body.Close()

	// Parse metrics
	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		fmt.Fprintf(w, `<div class="error-message">Error parsing metrics: %v</div>`, err)
		return
	}

	// Display metrics
	for name, mf := range metrics {
		fmt.Fprintf(w, `<div class="metric">`)
		fmt.Fprintf(w, `<div class="metric-name">%s</div>`, name)
		fmt.Fprintf(w, `<div class="metric-type">%s</div>`, mf.GetType().String())
		fmt.Fprintf(w, `<div class="metric-help">%s</div>`, mf.GetHelp())

		for _, m := range mf.GetMetric() {
			fmt.Fprintf(w, `<div class="metric-value">`)

			// Display labels if present
			if len(m.GetLabel()) > 0 {
				fmt.Fprintf(w, `<span class="metric-labels">`)
				fmt.Fprintf(w, "{")
				for i, label := range m.GetLabel() {
					if i > 0 {
						fmt.Fprintf(w, ", ")
					}
					fmt.Fprintf(w, "%s=%s", label.GetName(), label.GetValue())
				}
				fmt.Fprintf(w, "}</span><br>")
			}

			// Display value based on metric type
			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				fmt.Fprintf(w, "Value: %.2f", m.GetCounter().GetValue())
			case dto.MetricType_GAUGE:
				fmt.Fprintf(w, "Value: %.2f", m.GetGauge().GetValue())
			case dto.MetricType_HISTOGRAM:
				fmt.Fprintf(w, "Count: %d<br>Sum: %.2f",
					m.GetHistogram().GetSampleCount(),
					m.GetHistogram().GetSampleSum())
			}

			fmt.Fprintf(w, "</div>")
		}
		fmt.Fprintf(w, "</div>")
	}

	fmt.Fprintf(w, `
				</div>
			</div>
		</body>
		</html>
	`)
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

var DISCARD_METRICS = map[string]struct{}{
	"go_sched_gomaxprocs_threads":                {},
	"go_memstats_gc_sys_bytes":                   {},
	"go_gc_gogc_percent":                         {},
	"go_memstats_heap_idle_bytes":                {},
	"go_memstats_heap_released_bytes":            {},
	"go_memstats_stack_inuse_bytes":              {},
	"go_memstats_stack_sys_bytes":                {},
	"go_memstats_buck_hash_sys_bytes":            {},
	"go_memstats_mcache_inuse_bytes":             {},
	"go_memstats_mcache_sys_bytes":               {},
	"go_memstats_mspan_inuse_bytes":              {},
	"go_memstats_mspan_sys_bytes":                {},
	"go_memstats_other_sys_bytes":                {},
	"go_memstats_alloc_bytes":                    {},
	"go_memstats_alloc_bytes_total":              {},
	"go_memstats_sys_bytes":                      {},
	"go_gc_gomemlimit_bytes":                     {},
	"promhttp_metric_handler_requests_total":     {},
	"promhttp_metric_handler_requests_in_flight": {},
	"scalability_metrics":                        {},
	"go_memstats_heap_objects":                   {},
	"go_memstats_next_gc_bytes":                  {},
	"go_memstats_gc_cpu_fraction":                {},
	"go_memstats_heap_sys_bytes":                 {},
	"go_memstats_heap_inuse_bytes":               {},
	"go_memstats_last_gc_time_seconds":           {},
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
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			// Scrape coordinator metrics
			coordMetrics, err := collector.scrapeEndpoint("http://localhost:8080/metrics")
			filteredCoordinatorMetrics := make([]*dto.MetricFamily, 0)
			if err != nil {
				logger.Error(err)
			} else {
				// Filter out metrics that should be discarded
				for _, mf := range coordMetrics {
					shouldDiscard := false
					_, shouldDiscard = DISCARD_METRICS[mf.GetName()]
					if !shouldDiscard {
						filteredCoordinatorMetrics = append(filteredCoordinatorMetrics, mf)
					}
				}
				collector.updateRegistry(filteredCoordinatorMetrics)
				logger.Info("Updated coordinator metrics")
			}

			// Scrape worker metrics (assuming workers are on ports 8081, 8082, etc.)
			workerPorts := []string{"9200", "9201", "9202", "9203"} // Add more ports as needed
			workerMetricsCombined := make([]*dto.MetricFamily, 0)
			for i, port := range workerPorts {
				workerID := fmt.Sprintf("worker-%d", i+1)
				workerMetrics, err := collector.scrapeEndpoint(fmt.Sprintf("http://localhost:%s/metrics", port))
				if err != nil {
					logger.WithField("workerID", workerID).Error(err)
					continue
				} else {
					// Filter out metrics that should be discarded
					filteredWorkerMetrics := make([]*dto.MetricFamily, 0)
					for _, mf := range workerMetrics {
						shouldDiscard := false
						_, shouldDiscard = DISCARD_METRICS[mf.GetName()]
						if !shouldDiscard {
							// Add worker ID label to each metric
							for _, m := range mf.Metric {
								m.Label = append(m.Label, &dto.LabelPair{
									Name:  proto.String("worker_id"),
									Value: proto.String(workerID),
								})
							}
							filteredWorkerMetrics = append(filteredWorkerMetrics, mf)
						}
						workerMetricsCombined = append(workerMetricsCombined, filteredWorkerMetrics...)
					}
					// Combine coordinator and worker metrics
					logger.WithField("workerID", workerID).Info("Updated worker metrics")
				}

			}
			combinedFilteredMetrics := append(workerMetricsCombined, filteredCoordinatorMetrics...)
			collector.updateRegistry(combinedFilteredMetrics)
		}
	}()

	logger.Info("Metrics server starting on port 2112")
	if err := http.ListenAndServe(":2112", nil); err != nil {
		logger.Fatal("Error starting metrics server: ", err)
	}
}
