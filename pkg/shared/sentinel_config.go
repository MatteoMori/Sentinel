package shared

// ExtraLabel defines how to extract a label/annotation from a workload and expose it as a Prometheus label
type ExtraLabel struct {
	Type                string `mapstructure:"type"`                // "annotation" or "label" - where to extract from
	Key                 string `mapstructure:"key"`                 // The annotation/label key to extract (e.g., "sentinel.io/owner")
	TimeseriesLabelName string `mapstructure:"timeseriesLabelName"` // The name to use in the Prometheus metric (e.g., "owner")
}

type Config struct {
	NamespaceSelector map[string]string `mapstructure:"namespaceSelector"` // Label selector for namespaces to watch
	MetricsPort       string            `mapstructure:"metricsPort"`       // Port for Prometheus metrics endpoint
	Verbosity         int               `mapstructure:"verbosity"`         // Log verbosity level (0-2)
	ExtraLabels       []ExtraLabel      `mapstructure:"extraLabels"`       // Additional labels to extract from workloads
}
