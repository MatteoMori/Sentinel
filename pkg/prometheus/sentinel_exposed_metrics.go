/*
This is where we define the metrics exposed by Sentinel.
*/

package prometheus

import "github.com/prometheus/client_golang/prometheus"

/*
NOTES:
- A Gauge is a metric that represents a single numerical value that can go up or down
- A GaugeVec is a collection of Gauges, partitioned by labels (here: namespace and deployment).

METRICS Definition

 1. ServiceQualityScore:
    -> sentinel_service_quality_score{namespace="team-a", deployment="web-api"} 76

 2. ServiceQualityRuleScore:
    -> sentinel_service_quality_rule_score{namespace="team-a", deployment="web-api", rule="readiness_probe"} 10
    -> sentinel_service_quality_rule_score{namespace="team-a", deployment="web-api", rule="owner_label"} 0

 3. ServiceQualityRuleStatus:
    -> sentinel_service_quality_rule_status{namespace="team-a", deployment="web-api", rule="readiness_probe"} 1
    -> sentinel_service_quality_rule_status{namespace="team-a", deployment="web-api", rule="readiness_probe"} 0
*/

var (
	ServiceQualityScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_service_quality_score",
			Help: "Aggregated Service Quality Score for an application",
		},
		[]string{"namespace", "deployment"},
	)

	// Score of each individual rule. Labeled to filter by a few parameters
	ServiceQualityRuleScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_service_quality_rule_score",
			Help: "Individual Service Quality Rule Score for an application",
		},
		[]string{"namespace", "deployment", "rule"},
	)

	// Status of each individual rule. Labeled to filter by a few parameters
	ServiceQualityRuleStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sentinel_service_quality_rule_status",
			Help: "Individual Service Quality Rule Status for an application",
		},
		[]string{"namespace", "deployment", "rule"},
	)

	// TODO: NEW metrics - Expose the weight of a rule
	// TODO: NEW metrics - Expose the Total possible score of a rule
)
