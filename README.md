# 🛡️ Sentinel

**A Kubernetes controller that watches your cluster and exposes container image inventory as Prometheus metrics.**

Sentinel monitors Kubernetes workloads (Deployments, StatefulSets, DaemonSets, CronJobs) across labeled namespaces and tracks which container images are actually running in your cluster—all exposed through Prometheus metrics.

---

## 🎯 Why Sentinel?

Gain real-time visibility into your cluster's container image landscape. Perfect for:

- **Image Inventory Tracking** – Know exactly what's running, where, and when
- **Security & Compliance** – Monitor image versions across namespaces
- **Version Drift Detection** – Spot outdated or unauthorized images quickly
- **Audit & Governance** – Track image usage patterns over time

---

## ✨ Features

- 🎯 **Label-based namespace selection** – Control what Sentinel watches using Kubernetes labels
- 📊 **Prometheus-native** – Seamless integration with your existing observability stack
- ⚡ **Real-time updates** – Powered by Kubernetes informers for instant reconciliation
- 🔧 **Configurable** – YAML config, environment variables, or CLI flags
- 🪶 **Lightweight** – Built with Go and runs as a single deployment

---

## 🚀 Quick Start

### Prerequisites

- Kubernetes cluster (>= v1.28)
- `kubectl` configured
- KIND or Minikube for local testing

### Installation

1. **Deploy Sentinel to your cluster:**

```bash
kubectl apply -f manifests/install/sentinel.yaml
```

2. **Label the namespaces you want Sentinel to watch:**

```bash
kubectl label namespace my-namespace sentinel.io/controlled=enabled
```

3. **Access metrics:**

Sentinel exposes metrics on port `9090` at `/metrics`:

```bash
kubectl port-forward -n kube-system svc/sentinel-metrics 9090:9090
curl localhost:9090/metrics
```

---

## 📊 Metrics Exposed

| Metric Name | Description | Labels |
|-------------|-------------|--------|
| `sentinel_service_quality_score` | Aggregated quality score for a workload | `namespace`, `deployment` |
| `sentinel_service_quality_rule_score` | Individual rule score | `namespace`, `deployment`, `rule` |
| `sentinel_service_quality_rule_status` | Rule status (pass/fail) | `namespace`, `deployment`, `rule` |

---

## ⚙️ Configuration

Sentinel can be configured via:

1. **Config file** (`/etc/sentinel/sentinel.yaml`):

```yaml
NamespaceSelector:
  "sentinel.io/controlled": "enabled"
metricsPort: "9090"
verbosity: 1
```

2. **Environment variables**:

```bash
export NAMESPACESELECTOR__sentinel.io/controlled=enabled
export METRICSPORT=9090
export VERBOSITY=2
```

3. **CLI flags**:

```bash
sentinel start -v=2
```

---

## 🏗️ Architecture

Sentinel is built using:

- **[client-go](https://github.com/kubernetes/client-go)** – Direct Kubernetes API interaction
- **Informers** – Efficient, real-time workload watching
- **[Prometheus client](https://github.com/prometheus/client_golang)** – Native metrics exposition

The controller maintains an up-to-date view of all container images by watching resource changes and reconciling state continuously.

---

## 🛠️ Local Development

### Build and Run Locally

```bash
# Initialize Go module
go mod tidy

# Build the binary
go build -o sentinel

# Run locally (requires kubeconfig)
./sentinel start -v=2
```

### Test with KIND

```bash
# Build Docker image
docker build -t sentinel:latest .

# Load into KIND cluster
kind load docker-image sentinel:latest --name <cluster-name>

# Deploy
kubectl apply -f manifests/install/sentinel.yaml

# Create demo namespaces with labels
kubectl apply -f manifests/develop/ns.yaml
kubectl apply -f manifests/develop/demo-apps.yaml
```

---

## 🎓 Learning Go & Kubernetes?

This project is designed as a learning resource for:

- Building Kubernetes controllers with client-go
- Working with informers and shared informer factories
- Exposing Prometheus metrics from Go applications
- Structuring production-ready Go projects

Feel free to explore the code, open issues, or contribute!

---

## 🤝 Contributing

Contributions are welcome! Whether it's:

- 🐛 Bug reports
- 💡 Feature requests
- 📖 Documentation improvements
- 🔧 Code contributions

Open an issue or submit a pull request.

---

## 📜 License

This project is licensed under the MIT License.

---

## 🌟 Project Status

Sentinel is in **active development** as a learning project to master Go and Kubernetes controller patterns.

**Current capabilities:**
- ✅ Namespace watching with label selectors
- ✅ Deployment monitoring
- ✅ Prometheus metrics server
- 🚧 Image extraction (in progress)
- 🚧 StatefulSet/DaemonSet/CronJob support (planned)
- 🚧 Custom label enrichment (planned)

---

## 📚 Resources

- [Kubernetes Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
- [client-go Documentation](https://github.com/kubernetes/client-go)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Writing Kubernetes Controllers](https://github.com/kubernetes/sample-controller)

---

**Built with ❤️ and Go** | [Report an Issue](https://github.com/MatteoMori/sentinel/issues)
