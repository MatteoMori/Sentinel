/*
  For each managed namespace, we need to discover the applications we want to inspect
  These are the resources we are monitoring:
	- Deployments
	- Statefulsets
	- CronJobs
	- Daemonsets

  Logic:
	SENTINEL watches a list of Kubernetes namespaces (provided via a channel) and, for each namespace, starts a watcher that monitors k8s Resources in that namespace.
	If a namespace is removed from the list, the code stops watching the Resource in that namespace.

    SENTINEL listens for updates to the list of namespaces (nsChannel).
    -> For each namespace in the updated list:
       - If it’s new, it starts an Informer for that namespace.
       - If it’s no longer present, it stops the informer for that namespace.
*/

package sentinel

import (
	"log/slog"
	"strings"
	"sync"

	SentinelPrometheus "github.com/MatteoMori/sentinel/pkg/prometheus"
	SentinelShared "github.com/MatteoMori/sentinel/pkg/shared"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// NamespaceInformer keeps track of an informer and its stop channel for a namespace.
type NamespaceInformer struct {
	StopCh  chan struct{}
	Factory informers.SharedInformerFactory
}

// AppDiscovery starts an Informer for each namespace received on the channel.
// It manages the lifecycle of informers, starting them for new namespaces and stopping them for removed namespaces.
func AppDiscovery(
	clientset *kubernetes.Clientset,
	nsChannel <-chan []string,
	sentinelConfig SentinelShared.Config) {
	slog.Debug("Listening for namespace updates...")

	var mu sync.Mutex
	activeInformers := make(map[string]*NamespaceInformer)

	for namespaces := range nsChannel {
		slog.Debug("Received namespaces", slog.Any("Namespaces", strings.Join(namespaces, ", ")))

		// Build a set of current namespaces for quick lookup
		currentSet := make(map[string]struct{}, len(namespaces))
		for _, ns := range namespaces {
			currentSet[ns] = struct{}{}
		}

		mu.Lock()
		// Start informers for new namespaces
		for _, ns := range namespaces {
			if _, exists := activeInformers[ns]; !exists {

				slog.Debug("Starting Resource informers for namespace", slog.String("Namespace", ns))
				stopCh := make(chan struct{})
				factory := informers.NewSharedInformerFactoryWithOptions(
					clientset,
					0,
					informers.WithNamespace(ns),
				)
				nsCopy := ns

				// Currently observed K8s resources
				DeploymentInformer := factory.Apps().V1().Deployments().Informer()
				StatefulsetsInformer := factory.Apps().V1().StatefulSets().Informer()
				DaemonsetsInformer := factory.Apps().V1().DaemonSets().Informer()

				// TODO: Enable support for the remaining K8s kinds
				//CronjobsInformer := factory.Batch().V1().CronJobs().Informer()

				// TODO: Add support for Init Containers
				/*
					Sentinel - Observe Deployments
				*/
				DeploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj interface{}) {
						deploy := obj.(*appsv1.Deployment)
						handleWorkloadAdd("Deployment", nsCopy, deploy, deploy.Spec.Template.Spec.Containers, sentinelConfig.ExtraLabels)
					},
					UpdateFunc: func(oldObj, newObj interface{}) {
						oldDeploy := oldObj.(*appsv1.Deployment)
						newDeploy := newObj.(*appsv1.Deployment)

						/* Skip if no actual change in resource version ( spurious update )
						Informers can sometimes emit updates even if the underlying object's content hasn't changed, based on internal cache syncs.
						ResourceVersion is the best indicator here. */
						if oldDeploy.ResourceVersion == newDeploy.ResourceVersion {
							slog.Debug("Skipping spurious update (ResourceVersion unchanged)", slog.Any("resource version", newDeploy.ResourceVersion), slog.String("ns/deployment", newDeploy.Namespace+"/"+newDeploy.Name))
							return
						}

						/* Evaluate ONLY if the spec (generation) has changed.
						- If newDeploy.Generation > oldDeploy.Generation, it means the user/client has updated the desired state (e.g., changed image, replicas). This is a meaningful update.
						- If newDeploy.Generation == oldDeploy.Generation, it means ONLY the status has changed, which includes the initial reconciliation updates. This is NOT a meaningful update */
						handleWorkloadUpdate("Deployment", nsCopy, newDeploy, oldDeploy.Generation, newDeploy.Generation, newDeploy.Spec.Template.Spec.Containers, oldDeploy.Spec.Template.Spec.Containers, sentinelConfig.ExtraLabels)
					},
					DeleteFunc: func(obj interface{}) {
						if deploy, ok := obj.(*appsv1.Deployment); ok {
							handleWorkloadDelete("Deployment", nsCopy, deploy.Name, deploy.Spec.Template.Spec.Containers)
						}
					},
				})

				/*
					Sentinel - Observe Statefulsets
				*/
				StatefulsetsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj interface{}) {
						statefulset := obj.(*appsv1.StatefulSet)
						handleWorkloadAdd("StatefulSet", nsCopy, statefulset, statefulset.Spec.Template.Spec.Containers, sentinelConfig.ExtraLabels)
					},
					UpdateFunc: func(oldObj, newObj interface{}) {
						oldStatefulSet := oldObj.(*appsv1.StatefulSet)
						newStatefulSet := newObj.(*appsv1.StatefulSet)

						/* Skip if no actual change in resource version ( spurious update )
						Informers can sometimes emit updates even if the underlying object's content hasn't changed, based on internal cache syncs.
						ResourceVersion is the best indicator here. */
						if oldStatefulSet.ResourceVersion == newStatefulSet.ResourceVersion {
							slog.Debug("Skipping spurious update (ResourceVersion unchanged)", slog.Any("resource version", newStatefulSet.ResourceVersion), slog.String("ns/statefulset", newStatefulSet.Namespace+"/"+newStatefulSet.Name))
							return
						}

						/* Evaluate ONLY if the spec (generation) has changed.
						- If newStatefulSet.Generation > oldStatefulSet.Generation, it means the user/client has updated the desired state (e.g., changed image, replicas). This is a meaningful update.
						- If newStatefulSet.Generation == oldStatefulSet.Generation, it means ONLY the status has changed, which includes the initial reconciliation updates. This is NOT a meaningful update */
						handleWorkloadUpdate("StatefulSet", nsCopy, newStatefulSet, oldStatefulSet.Generation, newStatefulSet.Generation, newStatefulSet.Spec.Template.Spec.Containers, oldStatefulSet.Spec.Template.Spec.Containers, sentinelConfig.ExtraLabels)
					},
					DeleteFunc: func(obj interface{}) {
						if statefulset, ok := obj.(*appsv1.StatefulSet); ok {
							handleWorkloadDelete("StatefulSet", nsCopy, statefulset.Name, statefulset.Spec.Template.Spec.Containers)
						}
					},
				})

				/*
					Sentinel - Observe Daemonsets
				*/
				DaemonsetsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj interface{}) {
						daemonset := obj.(*appsv1.DaemonSet)
						handleWorkloadAdd("DaemonSet", nsCopy, daemonset, daemonset.Spec.Template.Spec.Containers, sentinelConfig.ExtraLabels)
					},
					UpdateFunc: func(oldObj, newObj interface{}) {
						oldDaemonSet := oldObj.(*appsv1.DaemonSet)
						newDaemonSet := newObj.(*appsv1.DaemonSet)

						/* Skip if no actual change in resource version ( spurious update )
						Informers can sometimes emit updates even if the underlying object's content hasn't changed, based on internal cache syncs.
						ResourceVersion is the best indicator here. */
						if oldDaemonSet.ResourceVersion == newDaemonSet.ResourceVersion {
							slog.Debug("Skipping spurious update (ResourceVersion unchanged)", slog.Any("resource version", newDaemonSet.ResourceVersion), slog.String("ns/daemonset", newDaemonSet.Namespace+"/"+newDaemonSet.Name))
							return
						}

						/* Evaluate ONLY if the spec (generation) has changed.
						- If newDaemonSet.Generation > oldDaemonSet.Generation, it means the user/client has updated the desired state (e.g., changed image, replicas). This is a meaningful update.
						- If newDaemonSet.Generation == oldDaemonSet.Generation, it means ONLY the status has changed, which includes the initial reconciliation updates. This is NOT a meaningful update */
						handleWorkloadUpdate("DaemonSet", nsCopy, newDaemonSet, oldDaemonSet.Generation, newDaemonSet.Generation, newDaemonSet.Spec.Template.Spec.Containers, oldDaemonSet.Spec.Template.Spec.Containers, sentinelConfig.ExtraLabels)
					},
					DeleteFunc: func(obj interface{}) {
						if daemonset, ok := obj.(*appsv1.DaemonSet); ok {
							handleWorkloadDelete("DaemonSet", nsCopy, daemonset.Name, daemonset.Spec.Template.Spec.Containers)
						}
					},
				})
				go factory.Start(stopCh)
				activeInformers[ns] = &NamespaceInformer{
					StopCh:  stopCh,
					Factory: factory,
				}
			}
		}

		// Stop informers for namespaces that are no longer present
		for ns, informer := range activeInformers {
			if _, stillPresent := currentSet[ns]; !stillPresent {
				slog.Info("Stopping Deployment informer for namespace", slog.String("Namespace", ns))
				close(informer.StopCh)
				delete(activeInformers, ns)
			}
		}
		mu.Unlock()
	}
}

func handleWorkloadAdd(resourceType, namespace string, workload metav1.Object, containers []corev1.Container, extraLabels []SentinelShared.ExtraLabel) {
	slog.Debug("New workload identified",
		slog.String("type", resourceType),
		slog.String("ns/name", namespace+"/"+workload.GetName()))

	// Extract extra label values from the workload
	extraLabelValues := extractExtraLabelValues(workload, extraLabels)

	// Process each container and set metrics
	for _, container := range containers {
		setContainerMetric(resourceType, namespace, workload.GetName(), container, extraLabelValues)
	}
}

func handleWorkloadUpdate(resourceType, namespace string, newWorkload metav1.Object, oldGen, newGen int64, newContainers []corev1.Container, oldContainers []corev1.Container, extraLabels []SentinelShared.ExtraLabel) {
	if newGen > oldGen {
		slog.Debug("Workload updated",
			slog.String("type", resourceType),
			slog.String("ns/name", namespace+"/"+newWorkload.GetName()))

		// Extract extra label values from the workload
		extraLabelValues := extractExtraLabelValues(newWorkload, extraLabels)

		// Build maps of old container images for comparison
		oldImages := make(map[string]string) // containerName -> image
		for _, container := range oldContainers {
			oldImages[container.Name] = container.Image
		}

		// Update metrics for all containers and detect changes
		for _, newContainer := range newContainers {
			setContainerMetric(resourceType, namespace, newWorkload.GetName(), newContainer, extraLabelValues)

			// Check if this container's image changed
			if oldImage, existed := oldImages[newContainer.Name]; existed && oldImage != newContainer.Image {
				// Image changed! Track it
				_, _, oldTag := parseImage(oldImage)
				_, _, newTag := parseImage(newContainer.Image)

				slog.Info("Image change detected",
					slog.String("workload", namespace+"/"+newWorkload.GetName()),
					slog.String("container", newContainer.Name),
					slog.String("old_tag", oldTag),
					slog.String("new_tag", newTag))

				// Increment the change counter
				SentinelPrometheus.SentinelImageChangesTotal.WithLabelValues(
					namespace,
					resourceType,
					newWorkload.GetName(),
					newContainer.Name,
					oldTag,
					newTag,
				).Inc()
			}
		}
	}
}

func handleWorkloadDelete(resourceType, namespace, name string, containers []corev1.Container) {
	slog.Debug("Workload deleted",
		slog.String("type", resourceType),
		slog.String("ns/name", namespace+"/"+name))

	// TODO: Delete Prometheus metrics for this workload
	// This is tricky because we need to track which label combinations exist
	// For now, metrics will persist (which is acceptable - they'll just stop updating)
	// A proper implementation would require maintaining a registry of active metrics
}

// setContainerMetric sets the Prometheus metric for a container image
// It parses the image string and combines base labels with extra labels
func setContainerMetric(workloadType, namespace, workloadName string, container corev1.Container, extraLabelValues []string) {
	// Parse the image into components
	registry, repository, tag := parseImage(container.Image)

	slog.Debug("Setting container metric",
		slog.String("ns/workload", namespace+"/"+workloadName),
		slog.String("container", container.Name),
		slog.String("image", container.Image),
		slog.String("registry", registry),
		slog.String("repository", repository),
		slog.String("tag", tag))

	// Build the complete label values slice
	// Order must match the order defined in BuildMetrics()
	labelValues := []string{
		namespace,
		workloadType,
		workloadName,
		container.Name,
		container.Image,
		registry,
		repository,
		tag,
	}

	// Append extra label values
	labelValues = append(labelValues, extraLabelValues...)

	// Set the metric (value is always 1 for info metrics)
	SentinelPrometheus.SentinelContainerImageInfo.WithLabelValues(labelValues...).Set(1)
}
