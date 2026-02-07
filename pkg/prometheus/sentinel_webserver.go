/*
This is a webserver listening on /metrics.

SCOPE:
- Expose Prometheus metrics coming from Sentinel
*/

package prometheus

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Init(metricsPort string) {
	// Register metrics
	prometheus.MustRegister(ServiceQualityScore)
	prometheus.MustRegister(ServiceQualityRuleScore)
	prometheus.MustRegister(ServiceQualityRuleStatus)

	// Start HTTP server in their Go Routine so that it does not block the main thread
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		slog.Debug("Starting Prometheus metrics endpoint on /metrics", slog.String("port", metricsPort))
		err := http.ListenAndServe(":"+metricsPort, nil)
		if err != nil {
			slog.Error("Metrics endpoint failed", slog.Any("error", err))
		}
	}()
}
