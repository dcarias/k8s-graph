package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kubegraph/config"
	"kubegraph/pkg/kubernetes/handlers"
	"kubegraph/pkg/logger"
	"kubegraph/pkg/neo4j"

	driverneo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// ResourceHandler defines the interface for handling Kubernetes resources
type ResourceHandler interface {
	// GetGVR returns the GroupVersionResource for this handler
	GetGVR() schema.GroupVersionResource
	// GetKind returns the kind of resource this handler manages
	GetKind() string
	// HandleCreate handles creation/update events
	HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error
	// HandleDelete handles deletion events
	HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error
}

// BaseHandler provides common functionality for resource handlers
type BaseHandler struct {
	gvr  schema.GroupVersionResource
	kind string
}

func (h *BaseHandler) GetGVR() schema.GroupVersionResource {
	return h.gvr
}

func (h *BaseHandler) GetKind() string {
	return h.kind
}

// Client represents the Kubernetes client
type Client struct {
	clientset       *kubernetes.Clientset
	dynamicClient   dynamic.Interface
	informerFactory dynamicinformer.DynamicSharedInformerFactory
	handlers        map[string]handlers.ResourceHandler
	config          *config.Config
}

// NewClient creates a new Kubernetes client
func NewClient(cfg *config.Config) (*Client, error) {
	var config *rest.Config
	var err error

	// Try config path from settings first
	if cfg.Kubernetes.ConfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.Kubernetes.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from specified path %s: %w", cfg.Kubernetes.ConfigPath, err)
		}
	} else {
		// Try standard kubeconfig locations
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		// Try to build config from kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			// If that fails, try in-cluster config
			config, err = rest.InClusterConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to create kubernetes config: no kubeconfig found in standard locations and not running in-cluster")
			}
		}
	}

	// Configure rate limiting and timeouts
	config.QPS = 50    // Increase QPS from default 5
	config.Burst = 100 // Increase Burst from default 10
	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create informer factory with longer resync period
	informerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 5*time.Minute)

	client := &Client{
		clientset:       clientset,
		dynamicClient:   dynamicClient,
		informerFactory: informerFactory,
		handlers:        make(map[string]handlers.ResourceHandler),
		config:          cfg,
	}

	// Register resource handlers
	client.registerHandlers()

	return client, nil
}

// registerHandlers registers all resource handlers
func (c *Client) registerHandlers() {
	resourceHandlers := []handlers.ResourceHandler{
		handlers.NewNodeHandler(c.config),
		handlers.NewPodHandler(c.clientset, c.config),
		handlers.NewServiceHandler(c.clientset, c.config),
		handlers.NewConfigMapHandler(c.config),
		handlers.NewSecretHandler(c.config),
		handlers.NewDeploymentHandler(c.clientset, c.config),
		handlers.NewReplicaSetHandler(c.clientset, c.config),
		handlers.NewStatefulSetHandler(c.clientset, c.config),
		handlers.NewDaemonSetHandler(c.clientset, c.config),
		handlers.NewJobHandler(c.clientset, c.config),
		handlers.NewCronJobHandler(c.clientset, c.config),
		handlers.NewPVHandler(c.config),
		handlers.NewPVCHandler(c.config),
		handlers.NewStorageClassHandler(c.config),
		handlers.NewNeo4jDatabaseHandler(c.config),
		handlers.NewNeo4jClusterHandler(c.config),
		handlers.NewNeo4jSingleInstanceHandler(c.config),
		handlers.NewNeo4jRoleHandler(c.config),
		handlers.NewIPAccessControlHandler(c.config),
		handlers.NewCustomEndpointHandler(c.config),
		handlers.NewBackupScheduleHandler(c.config),
		handlers.NewDomainNameHandler(c.config),
		handlers.NewNamespaceHandler(c.config),
		handlers.NewServiceAccountHandler(c.config),
		handlers.NewHorizontalPodAutoscalerHandler(c.config),
		handlers.NewVerticalPodAutoscalerHandler(c.config),
		handlers.NewPodDisruptionBudgetHandler(c.config),
		handlers.NewLimitRangeHandler(c.config),
		handlers.NewIngressHandler(c.config),
		handlers.NewEndpointsHandler(c.config),
		handlers.NewNetworkPolicyHandler(c.config),
	}
	if c.config.EventTTLDays > 0 {
		resourceHandlers = append(resourceHandlers, handlers.NewEventHandler(c.config))
	}

	for _, handler := range resourceHandlers {
		c.handlers[handler.GetKind()] = handler
	}
}

// StartWatching starts watching Kubernetes resources
func (c *Client) StartWatching(ctx context.Context, neo4jClient *neo4j.Client) error {
	logger.Info("Starting to watch Kubernetes resources...")

	// Set up informers for each handler
	informers := make([]cache.SharedInformer, 0, len(c.handlers))
	skippedHandlers := make([]string, 0)

	for _, handler := range c.handlers {
		// Create a new variable in this scope to avoid closure issues
		h := handler
		logger.Info("Setting up informer for resource type: %s", h.GetKind())

		// Check if the resource exists before setting up the informer
		gvr := h.GetGVR()
		var err error

		// For namespaced resources, try listing in default namespace
		// For cluster-scoped resources, list without namespace
		if c.isNamespacedResource(gvr) {
			_, err = c.dynamicClient.Resource(gvr).Namespace("default").List(ctx, metav1.ListOptions{Limit: 1})
		} else {
			_, err = c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{Limit: 1})
		}

		if err != nil {
			errorMsg := err.Error()
			logger.Debug("Resource check failed for %s (%s): %v", h.GetKind(), gvr.String(), err)
			if strings.Contains(errorMsg, "the server could not find the requested resource") ||
				strings.Contains(errorMsg, "not found") ||
				strings.Contains(errorMsg, "does not exist") ||
				strings.Contains(errorMsg, "forbidden") ||
				strings.Contains(errorMsg, "is forbidden") {
				// Check if this is a custom resource (non-core Kubernetes resource)
				if gvr.Group != "" && gvr.Group != "core" && gvr.Group != "v1" {
					logger.Info("Custom resource %s (%s) not available in cluster - this is normal if the corresponding CRD is not installed or RBAC permissions are missing", h.GetKind(), gvr.String())
				} else {
					logger.Warn("Resource %s (%s) not found in cluster or access forbidden, skipping informer setup", h.GetKind(), gvr.String())
				}
				skippedHandlers = append(skippedHandlers, h.GetKind())
				continue
			} else {
				logger.Warn("Failed to check if resource %s exists: %v", h.GetKind(), err)
				// Continue anyway, the informer might still work
			}
		}

		informer := c.informerFactory.ForResource(gvr).Informer()

		// Add backoff retry for event handlers
		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				logger.Debug("Received Add event for %s", h.GetKind())
				if err := h.HandleCreate(ctx, obj, neo4jClient); err != nil {
					if !isContextCanceled(err) {
						logger.Error("Error handling create event for %s: %v", h.GetKind(), err)
					}
				} else {
					logger.Debug("Successfully processed Add event for %s", h.GetKind())
				}
			},
			UpdateFunc: func(old, new interface{}) {
				logger.Debug("Received Update event for %s", h.GetKind())
				if err := h.HandleCreate(ctx, new, neo4jClient); err != nil {
					if !isContextCanceled(err) {
						logger.Error("Error handling update event for %s: %v", h.GetKind(), err)
					}
				} else {
					logger.Debug("Successfully processed Update event for %s", h.GetKind())
				}
			},
			DeleteFunc: func(obj interface{}) {
				logger.Debug("Received Delete event for %s", h.GetKind())
				if err := h.HandleDelete(ctx, obj, neo4jClient); err != nil {
					if !isContextCanceled(err) {
						logger.Error("Error handling delete event for %s: %v", h.GetKind(), err)
					}
				} else {
					logger.Debug("Successfully processed Delete event for %s", h.GetKind())
				}
			},
		})
		informers = append(informers, informer)
	}

	// Log summary of handler setup
	if len(skippedHandlers) > 0 {
		logger.Warn("Skipped %d handlers due to missing CRDs: %v", len(skippedHandlers), skippedHandlers)
	}
	logger.Info("Successfully set up %d informers", len(informers))

	// Start informers
	logger.Info("Starting informer factory...")
	c.informerFactory.Start(ctx.Done())

	// Wait for caches to sync with timeout
	logger.Info("Waiting for caches to sync...")
	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	for _, informer := range informers {
		if !cache.WaitForCacheSync(syncCtx.Done(), informer.HasSynced) {
			return fmt.Errorf("failed to sync cache for informer")
		}
	}
	logger.Info("All caches synced successfully")

	// Create a ticker to periodically check connections
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Monitor context and ticker
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("Context cancelled, stopping watch")
				return
			case <-ticker.C:
				// Check Neo4j connection with a simple query
				session := neo4jClient.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeWrite})
				_, err := session.Run(ctx, "RETURN 1", nil)
				if err != nil {
					logger.Warn("Neo4j connectivity check failed: %v", err)
				}
				session.Close(ctx)

				// Log informer status
				for _, handler := range c.handlers {
					informer := c.informerFactory.ForResource(handler.GetGVR()).Informer()
					logger.Debug("Informer status for %s - HasSynced: %v", handler.GetKind(), informer.HasSynced())
				}
			}
		}
	}()

	logger.Info("Watching for Kubernetes events...")
	<-ctx.Done()
	logger.Info("Stopping watch due to context cancellation")
	return nil
}

// isContextCanceled checks if the error is due to context cancellation
func isContextCanceled(err error) bool {
	if err == nil {
		return false
	}
	return err == context.Canceled || err == context.DeadlineExceeded || strings.Contains(err.Error(), "context canceled")
}

// isNamespacedResource determines if a resource is namespaced based on its GVR
func (c *Client) isNamespacedResource(gvr schema.GroupVersionResource) bool {
	// Core Kubernetes namespaced resources
	namespacedResources := map[string]bool{
		"pods":                     true,
		"services":                 true,
		"configmaps":               true,
		"secrets":                  true,
		"deployments":              true,
		"replicasets":              true,
		"statefulsets":             true,
		"daemonsets":               true,
		"jobs":                     true,
		"cronjobs":                 true,
		"persistentvolumeclaims":   true,
		"events":                   true,
		"namespaces":               false, // Namespaces are cluster-scoped
		"serviceaccounts":          true,
		"horizontalpodautoscalers": true,
		"verticalpodautoscalers":   true,
		"poddisruptionbudgets":     true,
		"limitranges":              true,
		"nodes":                    false, // Nodes are cluster-scoped
		"persistentvolumes":        false, // PVs are cluster-scoped
		"storageclasses":           false, // StorageClasses are cluster-scoped
		"ipaccesscontrols":         true,  // IPAccessControl is namespaced
		"customendpoints":          false, // CustomEndpoint is cluster-scoped
		"ingresses":                true,  // Ingresses are namespaced
		"endpoints":                true,  // Endpoints are namespaced
		"networkpolicies":          true,  // NetworkPolicies are namespaced
	}

	// Check if it's a known core resource
	if isNamespaced, exists := namespacedResources[gvr.Resource]; exists {
		return isNamespaced
	}

	// For custom resources, assume they are namespaced unless we know otherwise
	// Most custom resources are namespaced
	return true
}

// Convert unstructured object to typed object
func convertToTyped[T any](obj interface{}) (T, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		var zero T
		return zero, fmt.Errorf("object is not *unstructured.Unstructured")
	}

	var typedObj T
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &typedObj)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to convert unstructured to typed: %w", err)
	}

	return typedObj, nil
}

// handleResourceDelete is a helper function for deleting resources from Neo4j
func handleResourceDelete(ctx context.Context, resourceType, uid string, neo4jClient *neo4j.Client) error {
	query := fmt.Sprintf("MATCH (r:%s {uid: $uid}) DETACH DELETE r", resourceType)
	session := neo4jClient.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, query, map[string]interface{}{"uid": uid})
	return err
}

// handleNodeChange handles node changes
func (c *Client) handleNodeChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewNodeHandler(c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

// handleNodeDelete handles node deletions
func (c *Client) handleNodeDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewNodeHandler(c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

// handlePodChange handles pod changes
func (c *Client) handlePodChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewPodHandler(c.clientset, c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

// handlePodDelete handles pod deletions
func (c *Client) handlePodDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewPodHandler(c.clientset, c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

// handleServiceChange handles service changes
func (c *Client) handleServiceChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewServiceHandler(c.clientset, c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

// handleServiceDelete handles service deletions
func (c *Client) handleServiceDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewServiceHandler(c.clientset, c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handleConfigMapChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewConfigMapHandler(c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handleConfigMapDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewConfigMapHandler(c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handleSecretChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewSecretHandler(c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handleSecretDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewSecretHandler(c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handleDeploymentChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewDeploymentHandler(c.clientset, c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handleDeploymentDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewDeploymentHandler(c.clientset, c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handleStatefulSetChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewStatefulSetHandler(c.clientset, c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handleStatefulSetDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewStatefulSetHandler(c.clientset, c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handleDaemonSetChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewDaemonSetHandler(c.clientset, c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handleDaemonSetDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewDaemonSetHandler(c.clientset, c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handlePVChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewPVHandler(c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handlePVDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewPVHandler(c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

func (c *Client) handlePVCChange(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewPVCHandler(c.config)
	return handler.HandleCreate(ctx, obj, neo4jClient)
}

func (c *Client) handlePVCDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	handler := handlers.NewPVCHandler(c.config)
	return handler.HandleDelete(ctx, obj, neo4jClient)
}

// GetHandlers returns all registered resource handlers
func (c *Client) GetHandlers() map[string]handlers.ResourceHandler {
	return c.handlers
}
