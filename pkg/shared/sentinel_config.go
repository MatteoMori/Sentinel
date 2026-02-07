package shared

type Config struct {
	NamespaceSelector map[string]string `mapstructure:"NamespaceSelector"` // Label selector for namespaces to watch
	MetricsPort       string            `mapstructure:"metricsPort"`       // Port for Prometheus metrics endpoint
	Verbosity         int               `mapstructure:"verbosity"`         // Log verbosity level (0-2)
}
