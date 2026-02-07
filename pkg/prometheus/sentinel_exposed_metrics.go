/*
This is where we define the metrics exposed by Sentinel.
*/

package prometheus

import (
	"github.com/MatteoMori/sentinel/pkg/shared"
	"github.com/prometheus/client_golang/prometheus"
)

/*
NOTES:
- A Gauge is a metric that represents a single numerical value that can go up or down
- A GaugeVec is a collection of Gauges, partitioned by labels (here: workload_namespace and deployment).

METRICS Definition

 1. SentinelContainerImageInfo:
	-> sentinel_container_image_info{
		workload_namespace="prod",
		workload_type="Deployment",           // Deployment, StatefulSet, DaemonSet, CronJob
		workload_name="api-server",
		container_name="app",
		image="ghcr.io/myorg/myapp:v1.2.3",   // Full image string
		image_registry="ghcr.io",             // Parsed registry
		image_repository="myorg/myapp",       // Parsed repo
		image_tag="v1.2.3",                   // Parsed tag
		# Dynamic labels from extraLabels config are appended here
	  } 1


*/

var (
	/*
	 SentinelContainerImageInfo is built dynamically based on extraLabels configuration
	 It's initialized by calling BuildMetrics() at startup
	*/
	SentinelContainerImageInfo *prometheus.GaugeVec

	// SentinelImageChangesTotal tracks every time a container's image tag changes
	// This is a counter that increments whenever we detect an image update
	SentinelImageChangesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_image_changes_total",
			Help: "Total number of container image changes detected",
		},
		[]string{
			"workload_namespace",
			"workload_type",
			"workload_name",
			"container_name",
			"old_image_tag",
			"new_image_tag",
		},
	)
)

/*
BuildMetrics constructs the Prometheus metrics with dynamic labels based on configuration
This must be called before registering metrics with the Prometheus registry
*/
func BuildMetrics(extraLabels []shared.ExtraLabel) {
	// Base labels that are always present
	baseLabels := []string{
		"workload_namespace",
		"workload_type",
		"workload_name",
		"container_name",
		"image",
		"image_registry",
		"image_repository",
		"image_tag",
	}

	// Append extra label names from configuration
	for _, el := range extraLabels {
		baseLabels = append(baseLabels, el.TimeseriesLabelName) // comes from pkg/shared/sentinel_config.go
	}

	SentinelContainerImageInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_container_image_info",
			Help: "Information about container images used in workloads",
		},
		baseLabels,
	)
}
