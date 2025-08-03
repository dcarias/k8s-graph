package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"k8s-graph/config"
	"k8s-graph/pkg/httpserver"
	"k8s-graph/pkg/kubernetes"
	"k8s-graph/pkg/kubernetes/handlers"
	"k8s-graph/pkg/logger"
	"k8s-graph/pkg/neo4j"

	"github.com/google/uuid"
	driverneo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// getEnvBool gets a boolean value from environment variable
func getEnvBool(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		val = strings.TrimSpace(val)
		if strings.ToLower(val) == "true" || val == "1" {
			return true
		}
		if strings.ToLower(val) == "false" || val == "0" {
			return false
		}
	}
	return defaultValue
}

// getEnvInt gets an integer value from environment variable
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		val = strings.TrimSpace(val)
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func main() {
	cfg := config.NewConfig()

	// Command line flags
	var kubeconfig string
	var clusterName string
	var neo4jURI string
	var neo4jUsername string
	var neo4jPassword string
	var httpEnabled bool
	var httpPort int
	var logLevel string
	var eventTTLDays int

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (uses in-cluster config if empty)")
	flag.StringVar(&clusterName, "cluster-name", "default", "Name of the Kubernetes cluster")
	flag.StringVar(&neo4jURI, "neo4j-uri", "neo4j://localhost:7687", "Neo4j database URI")
	flag.StringVar(&neo4jUsername, "neo4j-username", "neo4j", "Neo4j username")
	flag.StringVar(&neo4jPassword, "neo4j-password", "password", "Neo4j password")
	flag.BoolVar(&httpEnabled, "http-enabled", true, "Enable HTTP server for status")
	flag.IntVar(&httpPort, "http-port", 8080, "HTTP server port")
	flag.StringVar(&logLevel, "log-level", "INFO", "Log level: DEBUG, INFO, WARN, ERROR")
	flag.IntVar(&eventTTLDays, "event-ttl-days", 7, "Number of days to retain Kubernetes events (0 disables event handling)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "k8s-graph - Kubernetes Resource Graph Database\n\n")
		fmt.Fprintf(os.Stderr, "A tool for synchronizing Kubernetes resources to Neo4j graph database.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Connect to local Neo4j\n")
		fmt.Fprintf(os.Stderr, "  %s --cluster-name=my-cluster\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Connect to remote Neo4j\n")
		fmt.Fprintf(os.Stderr, "  %s --neo4j-uri=neo4j://remote:7687 --neo4j-username=user --neo4j-password=pass\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Disable event monitoring for performance\n")
		fmt.Fprintf(os.Stderr, "  %s --event-ttl-days=0\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Environment Variables:\n")
		fmt.Fprintf(os.Stderr, "  KUBECONFIG       - Path to kubeconfig file\n")
		fmt.Fprintf(os.Stderr, "  CLUSTER_NAME     - Kubernetes cluster name\n")
		fmt.Fprintf(os.Stderr, "  NEO4J_URI        - Neo4j database URI\n")
		fmt.Fprintf(os.Stderr, "  NEO4J_USERNAME   - Neo4j username\n")
		fmt.Fprintf(os.Stderr, "  NEO4J_PASSWORD   - Neo4j password\n")
		fmt.Fprintf(os.Stderr, "  LOG_LEVEL        - Log level\n")
		fmt.Fprintf(os.Stderr, "  HTTP_ENABLED     - Enable HTTP server (true/false)\n")
		fmt.Fprintf(os.Stderr, "  HTTP_PORT        - HTTP server port\n\n")
		fmt.Fprintf(os.Stderr, "Supported Resources:\n")
		fmt.Fprintf(os.Stderr, "  • Pods: Pod lifecycle and relationships\n")
		fmt.Fprintf(os.Stderr, "  • Deployments: Deployment configurations\n")
		fmt.Fprintf(os.Stderr, "  • Services: Service endpoints and selectors\n")
		fmt.Fprintf(os.Stderr, "  • ConfigMaps: Configuration data\n")
		fmt.Fprintf(os.Stderr, "  • Secrets: Secret metadata (data excluded)\n")
		fmt.Fprintf(os.Stderr, "  • Events: Kubernetes Events (with TTL)\n")
		fmt.Fprintf(os.Stderr, "  • Plus networking, storage, RBAC, and autoscaling resources\n\n")
	}

	flag.Parse()

	// Override with environment variables
	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfig = envKubeconfig
	}
	if envClusterName := os.Getenv("CLUSTER_NAME"); envClusterName != "" {
		clusterName = envClusterName
	}
	if envNeo4jURI := os.Getenv("NEO4J_URI"); envNeo4jURI != "" {
		neo4jURI = envNeo4jURI
	}
	if envNeo4jUsername := os.Getenv("NEO4J_USERNAME"); envNeo4jUsername != "" {
		neo4jUsername = envNeo4jUsername
	}
	if envNeo4jPassword := os.Getenv("NEO4J_PASSWORD"); envNeo4jPassword != "" {
		neo4jPassword = envNeo4jPassword
	}
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		logLevel = envLogLevel
	}

	httpEnabled = getEnvBool("HTTP_ENABLED", httpEnabled)
	httpPort = getEnvInt("HTTP_PORT", httpPort)

	// Update config
	cfg.Kubernetes.ConfigPath = kubeconfig
	cfg.Kubernetes.ClusterName = clusterName
	cfg.Neo4j.URI = neo4jURI
	cfg.Neo4j.Username = neo4jUsername
	cfg.Neo4j.Password = neo4jPassword
	cfg.HTTP.Enabled = httpEnabled
	cfg.HTTP.Port = httpPort
	cfg.EventTTLDays = eventTTLDays
	cfg.InstanceHash = uuid.New().String()

	// Initialize logger
	logger.SetLevel(logLevel)
	logger.Info("Starting k8s-graph...")
	logger.Info("Cluster: %s", clusterName)
	logger.Info("Neo4j URI: %s", neo4jURI)
	logger.Info("Instance Hash: %s", cfg.InstanceHash)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Neo4j client
	neo4jClient, err := neo4j.NewClient(cfg)
	if err != nil {
		logger.Error("Failed to create Neo4j client: %v", err)
		os.Exit(1)
	}
	defer neo4jClient.Close(ctx)

	logger.Info("Connected to Neo4j database")

	// Create Kubernetes client
	kubernetesClient, err := kubernetes.NewClient(cfg)
	if err != nil {
		logger.Error("Failed to create Kubernetes client: %v", err)
		os.Exit(1)
	}

	// Create resource handlers for standard Kubernetes resources
	var resourceHandlers []handlers.ResourceHandler
	
	// Core workload resources
	resourceHandlers = append(resourceHandlers, handlers.NewPodHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewDeploymentHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewReplicaSetHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewDaemonSetHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewStatefulSetHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewJobHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewCronJobHandler(cfg))

	// Services and networking
	resourceHandlers = append(resourceHandlers, handlers.NewServiceHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewEndpointsHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewIngressHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewNetworkPolicyHandler(cfg))

	// Configuration and storage
	resourceHandlers = append(resourceHandlers, handlers.NewConfigMapHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewSecretHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewPVHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewPVCHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewStorageClassHandler(cfg))

	// RBAC and policies
	resourceHandlers = append(resourceHandlers, handlers.NewServiceAccountHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewLimitRangeHandler(cfg))

	// Cluster resources
	resourceHandlers = append(resourceHandlers, handlers.NewNodeHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewNamespaceHandler(cfg))

	// Autoscaling
	resourceHandlers = append(resourceHandlers, handlers.NewHPAHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewVPAHandler(cfg))
	resourceHandlers = append(resourceHandlers, handlers.NewPDBHandler(cfg))

	// Events (if enabled)
	if cfg.EventTTLDays > 0 {
		resourceHandlers = append(resourceHandlers, handlers.NewEventHandler(cfg))
		logger.Info("Event monitoring enabled (TTL: %d days)", cfg.EventTTLDays)
	} else {
		logger.Info("Event monitoring disabled")
	}

	logger.Info("Registered %d resource handlers", len(resourceHandlers))

	// Start background cleanup process
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Clean up duplicate clusters with same name but different hashes
				if err := neo4jClient.CleanupDuplicateClusters(ctx, cfg.Kubernetes.ClusterName, cfg.InstanceHash); err != nil {
					logger.Error("[CLEANUP] Failed to cleanup duplicate clusters: %v", err)
				} else {
					logger.Debug("[CLEANUP] Duplicate cluster cleanup completed")
				}
				// Prune expired events if enabled
				if cfg.EventTTLDays > 0 {
					err := handlers.PruneExpiredEvents(ctx, neo4jClient, cfg.EventTTLDays)
					if err != nil {
						logger.Error("[EVENT PRUNE] Failed to prune expired events: %v", err)
					} else {
						logger.Debug("[EVENT PRUNE] Expired events pruned (TTL=%d days)", cfg.EventTTLDays)
					}
				}
			}
		}
	}()

	// Start watching resources
	err = kubernetesClient.StartWatching(ctx, resourceHandlers, neo4jClient)
	if err != nil {
		logger.Error("Failed to start watching resources: %v", err)
		os.Exit(1)
	}

	// Start HTTP server if enabled
	if cfg.HTTP.Enabled {
		server := httpserver.NewServer(cfg)
		go func() {
			if err := server.Start(); err != nil {
				logger.Error("HTTP server error: %v", err)
			}
		}()
		logger.Info("HTTP server started on port %d", cfg.HTTP.Port)
	}

	logger.Info("k8s-graph is running. Press Ctrl+C to stop.")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down...")
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)
	logger.Info("Shutdown complete")
}