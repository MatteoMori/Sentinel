/*
This is the Prometheus client implementation used when running Prometheus queries.

SCOPE:
- Evaluate Prometheus based queries
*/

package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient is an interface for running Prometheus queries.
type PrometheusClient interface {
	Query(ctx context.Context, query string) (model.Value, error)
}

// PromClient implements the PrometheusClient interface using the Prometheus Go client library.
type PromClient struct {
	api v1.API
}

// NewPrometheusClient returns a Prometheus client using the given base URL
func NewPrometheusClient(baseURL string) (PrometheusClient, error) {
	client, err := api.NewClient(api.Config{Address: baseURL})
	if err != nil {
		return nil, err
	}
	return &PromClient{api: v1.NewAPI(client)}, nil // RETURNs the Interface itself
}

// Query runs a PromQL query - Uses the Interface PrometheusClient
func (c *PromClient) Query(ctx context.Context, query string) (model.Value, error) {
	value, _, err := c.api.Query(ctx, query, time.Now())
	return value, err
}
