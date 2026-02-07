# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Sentinel is a Kubernetes controller that tracks container images across workloads and exposes them as Prometheus metrics. It monitors Deployments, StatefulSets, and DaemonSets (CronJobs planned) and provides:

1. **Container image inventory** via `sentinel_container_image_info` metric
2. **Image change tracking** via `sentinel_image_changes_total` counter
3. **Dynamic label enrichment** - extract workload annotations/labels into Prometheus labels

## Build & Deploy Commands

```bash
# Build
make build                    # Build Go binary
make docker                   # Build Docker image
make deploy                   # Build + load to KIND + deploy to k8s (uses cluster "homelab")

# Run locally (requires kubeconfig)
make run                      # Build + run with -v=2

# Test
make test                     # Run all tests (currently no tests exist)

# Dependencies
make deps                     # go mod tidy
```

**Manual deployment:**
```bash
# Build and load into custom KIND cluster
docker build -t sentinel:latest .
kind load docker-image sentinel:latest --name <cluster-name>
kubectl apply -f manifests/install/sentinel.yaml

# Deploy demo workloads
kubectl apply -f manifests/develop/demo-app-1.yaml
kubectl apply -f manifests/develop/demo-app-2.yaml

# Verify
kubectl port-forward -n kube-system svc/sentinel-metrics 9090:9090
curl -s localhost:9090/metrics | grep sentinel_
```

## Architecture

### Control Flow

```
main.go
  └─> cmd/sentinel/root.go (Cobra CLI)
       └─> cmd/sentinel/start.go (loads config via Viper)
            └─> pkg/sentinel/start.go
                 ├─> pkg/prometheus/sentinel_webserver.go (Init metrics + HTTP server)
                 │    └─> pkg/prometheus/sentinel_exposed_metrics.go (BuildMetrics with dynamic labels)
                 │
                 ├─> NamespaceWatcher() (watches namespaces with label selector)
                 │    └─> sends []string of namespace names via channel
                 │
                 └─> AppDiscovery() (consumes namespace channel)
                      └─> pkg/sentinel/app_discovery.go
                           ├─> Creates SharedInformerFactory per namespace
                           ├─> Watches Deployments, StatefulSets, DaemonSets
                           └─> On events: handleWorkloadAdd/Update/Delete
                                └─> setContainerMetric() (sets Prometheus metrics)
```

### Key Concepts

**Namespace Watching:**
- `NamespaceWatcher()` monitors namespaces matching `Config.NamespaceSelector` (default: `sentinel.io/controlled=enabled`)
- Sends updated namespace list via channel whenever namespaces are labeled/unlabeled
- `AppDiscovery()` consumes this channel and starts/stops informers per namespace

**Informer Lifecycle:**
- Each watched namespace gets its own `SharedInformerFactory`
- Informers watch Deployments and trigger event handlers
- When namespace is unlabeled, informer is stopped via `close(stopCh)`

**Dynamic Prometheus Labels:**
- Metrics are built at startup via `BuildMetrics(extraLabels)`
- Base labels (workload_namespace, workload_type, etc.) + dynamic labels from `Config.ExtraLabels`
- Prometheus requires all label names defined at registration time (can't add labels later)
- **Label naming:** Uses `workload_namespace` instead of `namespace` to avoid collision with Prometheus ServiceMonitor auto-labels

**Image Change Detection:**
- `handleWorkloadUpdate()` compares old vs new containers
- If `container.Image` changed, increments `sentinel_image_changes_total{old_tag="...", new_tag="..."}`
- Uses `parseImage()` helper to extract registry/repo/tag

## File Organization

```
cmd/sentinel/          - CLI definition (Cobra)
  root.go              - Root command
  start.go             - "start" subcommand + Viper config loading

pkg/shared/            - Shared types
  sentinel_config.go   - Config and ExtraLabel structs

pkg/prometheus/        - Metrics
  sentinel_exposed_metrics.go  - Metric definitions + BuildMetrics()
  sentinel_webserver.go         - HTTP server on :9090/metrics

pkg/sentinel/          - Controller logic
  start.go             - Main controller entrypoint
  app_discovery.go     - Per-namespace informers + event handlers
  helpers.go           - Utilities (parseImage, extractExtraLabelValues, etc.)

manifests/
  install/             - Production deployment (ConfigMap, Deployment, RBAC)
  develop/             - Demo workloads for testing

dashboard/
  grafana.json         - Pre-built Grafana dashboard
```

## Configuration

Configuration is loaded via Viper with this precedence (highest to lowest):
1. Environment variables (e.g., `METRICSPORT`, `VERBOSITY`)
2. Config file at `/etc/sentinel/sentinel.yaml`
3. Defaults in `cmd/sentinel/start.go`

**Example config:**
```yaml
namespaceSelector:
  "sentinel.io/controlled": "enabled"
metricsPort: "9090"
verbosity: 2

extraLabels:
  - type: "annotation"
    key: "sentinel.io/owner"
    timeseriesLabelName: "owner"
  - type: "label"
    key: "environment"
    timeseriesLabelName: "env"
```

## Prometheus Metrics Behavior

### Info Metrics (sentinel_container_image_info)

- Always has value `1` (info pattern)
- When image tag changes: old time series stops being reported, new time series starts
- Prometheus caches old series briefly (5-15min) before expiring them
- **Empty labels:** If annotation/label doesn't exist on workload, metric label is `""`

### Counter Metrics (sentinel_image_changes_total)

- Increments on every image tag change
- **Important:** Counter is created on-demand when first change detected
- Prometheus sees counter appear at value `1` (not `0`→`1`), so `increase()` over short windows may return `0`
- Use `increase(sentinel_image_changes_total[1h])` or longer windows for reliable detection

## Workload Type Support

**Currently Supported:** Deployments, StatefulSets, DaemonSets
**Planned:** CronJobs

### Implementation Pattern for Workload Types

All workload handlers use polymorphism via `metav1.Object` interface:
- `handleWorkloadAdd(resourceType string, namespace string, workload metav1.Object, ...)`
- `handleWorkloadUpdate(resourceType string, namespace string, newWorkload metav1.Object, ...)`
- `handleWorkloadDelete(resourceType string, namespace string, name string, ...)`

This allows a single set of handlers to work with Deployment/StatefulSet/DaemonSet/etc.

**Key:** Use `.GetName()`, `.GetAnnotations()`, `.GetLabels()` methods (not direct field access like `.Name`)

## Important Implementation Details

### Image Parsing (helpers.go:parseImage)
- Handles full registry URLs (ghcr.io, quay.io, etc.)
- Defaults to `docker.io` if no registry in image string
- Detects registry vs namespace by looking for `.` or `:` in first path component
- Default tag is `latest` if not specified

### Change Detection (app_discovery.go:handleWorkloadUpdate)
- Only processes updates where `newGen > oldGen` (spec changes, not status changes)
- Skips spurious updates where `ResourceVersion` unchanged
- Compares old vs new container images by building a map of `containerName -> image`
- **Limitation:** If container is added/removed, no change event (only updates to existing containers)

### Metric Deletion
- Currently NOT implemented (see TODO at app_discovery.go:~200)
- Deleted workloads leave metrics in Prometheus until scrape timeout
- To implement: would need to track active metrics and call `.Delete()` on GaugeVec

## Development Workflow

1. **Make code changes**
2. **Build:** `make build` (or `go build -o sentinel`)
3. **Test locally:** `make run` (requires kubeconfig pointing to cluster)
4. **Deploy to KIND:** `make deploy` (builds Docker + loads to cluster "homelab")
5. **Check logs:** `kubectl logs -n kube-system -l app=sentinel-controller -f`
6. **Verify metrics:** Port-forward and curl `/metrics`

## Prometheus ServiceMonitor

`manifests/install/servicemonitor.yaml` configures Prometheus Operator scraping with `metricRelabelings`:

```yaml
metricRelabelings:
  - action: labeldrop
    regex: pod           # Drop Prometheus auto-labels
  - action: labeldrop
    regex: endpoint
  - action: labeldrop
    regex: instance
  - action: labeldrop
    regex: service
  - action: labeldrop
    regex: namespace     # ServiceMonitor adds namespace="kube-system"
```

**Why:** Prometheus ServiceMonitor automatically adds labels (`pod`, `endpoint`, `instance`, `service`, `namespace`) when scraping. We drop these to keep metrics clean since they're not meaningful for Sentinel's use case.

**Note:** Changes to `metricRelabelings` only affect NEW samples. Old time series with previous labels persist in Prometheus TSDB until retention expires.

## Grafana Dashboard

Pre-built dashboard at `dashboard/grafana.json` includes:
- Overview stats (tracked containers, workloads, changes, `:latest` usage)
- Image inventory table with color-coded tags
- Registry distribution pie chart
- Image changes log (table format works better than graphs for counter metrics)

Import into Grafana via UI → Dashboards → Import → Upload JSON file.
