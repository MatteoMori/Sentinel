package sentinel

import (
	"log/slog"
	"os"
	"strings"

	SentinelShared "github.com/MatteoMori/sentinel/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// namespaceMatchesSelector checks if a namespace has all the required labels with correct values
func namespaceMatchesSelector(ns *v1.Namespace, selector map[string]string) bool {
	for key, value := range selector {
		if ns.Labels[key] != value {
			return false
		}
	}
	return true
}

// setupLogging configures the logging level based on the verbosity setting.
func setupLogging(verbosity int) {
	var level slog.Level
	switch verbosity {
	case 0:
		level = slog.LevelInfo
	case 1:
		level = slog.LevelWarn
	case 2:
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	slog.SetDefault(slog.New(handler))
}

func slicePurge(slice []string, item string) []string {
	// This function removes an item from a slice if it exists.
	// It returns a new slice without the specified item.
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice // Return the original slice if the item was not found
}

/*
extractExtraLabelValues extracts label/annotation values from a Kubernetes object based on configuration
- Returns a slice of values in the same order as the extraLabels config
- If a label/annotation is not found, an empty string is used
*/
func extractExtraLabelValues(obj metav1.Object, extraLabels []SentinelShared.ExtraLabel) []string {
	values := make([]string, len(extraLabels))

	for i, extractor := range extraLabels {
		var value string

		switch extractor.Type {
		case "annotation":
			if obj.GetAnnotations() != nil {
				value = obj.GetAnnotations()[extractor.Key]
			}
		case "label":
			if obj.GetLabels() != nil {
				value = obj.GetLabels()[extractor.Key]
			}
		default:
			slog.Warn("Unknown extraLabel type, skipping",
				slog.String("type", extractor.Type),
				slog.String("key", extractor.Key))
		}

		// Use empty string if not found (Prometheus requires all series to have same label set)
		values[i] = value
	}

	return values
}

// parseImage splits a container image string into registry, repository, and tag components
// Example: "ghcr.io/myorg/myapp:v1.2.3" -> ("ghcr.io", "myorg/myapp", "v1.2.3")
func parseImage(image string) (registry, repository, tag string) {
	// Default tag if not specified
	tag = "latest"

	// Split by tag separator ':'
	parts := strings.SplitN(image, ":", 2)
	imagePath := parts[0]
	if len(parts) == 2 {
		tag = parts[1]
	}

	// Split image path into registry and repository
	// If there's no '/', assume it's docker.io (Docker Hub)
	pathParts := strings.SplitN(imagePath, "/", 2)
	if len(pathParts) == 1 {
		// No registry specified, default to docker.io
		registry = "docker.io"
		repository = pathParts[0]
	} else {
		// Check if first part looks like a registry (has '.' or ':' or is 'localhost')
		if strings.Contains(pathParts[0], ".") || strings.Contains(pathParts[0], ":") || pathParts[0] == "localhost" {
			registry = pathParts[0]
			repository = pathParts[1]
		} else {
			// First part is namespace, not registry (e.g., "library/nginx")
			registry = "docker.io"
			repository = imagePath
		}
	}

	return registry, repository, tag
}
