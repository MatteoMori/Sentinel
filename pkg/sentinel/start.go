package sentinel

import (
	"context"
	"log"
	"log/slog"
	"slices"

	SentinelPrometheus "github.com/MatteoMori/sentinel/pkg/prometheus"
	SentinelShared "github.com/MatteoMori/sentinel/pkg/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func Start(Config SentinelShared.Config) {
	setupLogging(Config.Verbosity)
	SentinelPrometheus.Init(Config.MetricsPort, Config.ExtraLabels)

	slog.Info("Starting Sentinel controller")
	slog.Debug("Loaded Sentinel Config", slog.Any("Sentinel Config", Config))
	// Initialize the clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("Failed to initialize clientset", slog.Any("error", err))
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create Kubernetes clientset", slog.Any("error", err))
		return
	}

	// Monitor the K8s cluster for new namespaces matching the label and return a channel to use after.
	nsChannel := NamespaceWatcher(clientset, Config.NamespaceSelector) // nsChannel will be used later by ServiceDiscovery
	AppDiscovery(clientset, nsChannel, Config)

	println("WE ARE DONE FOR NOW")
}

/*
Monitor the K8s cluster for namespaces matching the Sentinel label selector MAP.
- Return: a Channel
*/
func NamespaceWatcher(clientset *kubernetes.Clientset, NamespaceSelector map[string]string) chan []string {
	/*
	  Build label selector string from map
	  You need to build this string: "sentinel.io/controlled=enabled"
	  If the map had multiple entries, you'd need: "key1=value1,key2=value2"
	*/
	var labelSelector string
	for key, value := range NamespaceSelector {
		if labelSelector != "" {
			labelSelector += "," // Add comma separator for multiple labels
		}
		labelSelector += key + "=" + value
	}

	// Start by getting a list of the existing namespaces containing the Sentinel label selector and the proper value
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err != nil {
		slog.Error("Failed to initialize Namespacewatcher", slog.Any("error", err))
		return nil
	}

	var initialNamespaces []string
	for i := range namespaces.Items {
		initialNamespaces = append(initialNamespaces, namespaces.Items[i].Name)
	}
	slog.Debug("Initial namespaces", slog.Any("Namespaces", initialNamespaces))

	// Open a channel and send the initial namespace list
	nsChannel := make(chan []string, 1)
	nsChannel <- initialNamespaces

	// Start a watcher and monitor for namespace Events
	factory := informers.NewSharedInformerFactory(clientset, 0)
	namespaceInformer := factory.Core().V1().Namespaces().Informer()

	// Define event handler for namespace events - The first time, this will get a list of all namespaces. Then only the new ones.
	namespaceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{

		// Check for New Namespaces
		AddFunc: func(obj interface{}) {
			// Get the newly created namespace
			namespace := obj.(*v1.Namespace)

			// Is this a new namespace + it contains the Sentinel label selector?
			if !slices.Contains(initialNamespaces, namespace.Name) && namespaceMatchesSelector(namespace, NamespaceSelector) {
				slog.Debug("A new namespace to monitor has been identified", slog.String("Namespaces", namespace.Name))

				// Append the new namespace name to the list
				initialNamespaces = append(initialNamespaces, namespace.Name)
				slog.Debug("Current list of monitored namespaces", slog.Any("Namespaces", initialNamespaces))
				select {
				case nsChannel <- initialNamespaces:
					// Successfully sent the updated list
				default:
					// Channel is full, handle this gracefully (e.g., log a warning)
					slog.Error("Failed to send updated namespace list to channel")
				}
			}
		},

		// Check for Events updating existing namespaces
		// - Has the Sentinel label selector being added, modified or removed?
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNs := oldObj.(*v1.Namespace)
			newNs := newObj.(*v1.Namespace)

			oldMatches := namespaceMatchesSelector(oldNs, NamespaceSelector)
			newMatches := namespaceMatchesSelector(newNs, NamespaceSelector)

			// The Sentinel label selector has been added or simply enabled
			if newMatches && !oldMatches {
				slog.Debug("Sentinel label selector has been added or enabled on namespace", slog.String("Namespaces", newNs.Name))
				initialNamespaces = append(initialNamespaces, newNs.Name)
				slog.Debug("Current list of monitored namespaces", slog.Any("Namespaces", initialNamespaces))

			} else if !newMatches && oldMatches { // The Sentinel label selector have been disabled or removed
				slog.Debug("Sentinel label selector has been disabled or removed on namespace", slog.String("Namespaces", newNs.Name))

				// Logic to remove the namespace from the initialNamespaces slice
				initialNamespaces = slicePurge(initialNamespaces, newNs.Name)
				slog.Debug("Current list of monitored namespaces", slog.Any("Namespaces", initialNamespaces))

			}
			select {
			case nsChannel <- initialNamespaces:
				// Successfully sent the updated list
			default:
				// Channel is full, handle this gracefully (e.g., log a warning)
				slog.Error("Failed to send updated namespace list to channel")
			}
		},

		// Check if a monitored namespace has been deleted
		DeleteFunc: func(obj interface{}) {
			namespace := obj.(*v1.Namespace)
			if slices.Contains(initialNamespaces, namespace.Name) && namespaceMatchesSelector(namespace, NamespaceSelector) {
				slog.Debug("Namespace deleted", slog.Any("Namespaces", namespace.Name))
				initialNamespaces = slicePurge(initialNamespaces, namespace.Name) // Remove the namespace from the initialNamespaces slice
				slog.Debug("Current list of monitored namespaces", slog.Any("Namespaces", initialNamespaces))
				select {
				case nsChannel <- initialNamespaces:
					// Successfully sent the updated list
				default:
					// Channel is full, handle this gracefully (e.g., log a warning)
					log.Println("[NamespaceWatcher] Failed to send updated namespace list to channel")
				}
			}
		},
	})

	// Start the namespace informer (runs in a separate goroutine). The channel is needed for the Sync below
	NsInformerCh := make(chan struct{})
	slog.Info("Starting namespace informer")
	go factory.Start(NsInformerCh)

	// Wait for the informer's cache to sync with the API server
	if !cache.WaitForCacheSync(NsInformerCh, namespaceInformer.HasSynced) {
		slog.Error("Error waiting for namespace informer caches to sync")
	}

	return nsChannel
}
