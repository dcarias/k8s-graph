package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"kubegraph/config"
	"kubegraph/pkg/kubernetes"
	"kubegraph/pkg/logger"
	"kubegraph/pkg/neo4j"
	"kubegraph/pkg/version"

	driverneo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server
type Server struct {
	config      *config.Config
	k8sClient   *kubernetes.Client
	neo4jClient *neo4j.Client
	server      *http.Server
	startTime   time.Time
}

// InfoResponse represents the response for the /info endpoint
type InfoResponse struct {
	Application   string                 `json:"application"`
	Version       string                 `json:"version"`
	GitCommit     string                 `json:"gitCommit"`
	GitBranch     string                 `json:"gitBranch"`
	StartTime     time.Time              `json:"startTime"`
	Uptime        string                 `json:"uptime"`
	ClusterName   string                 `json:"clusterName"`
	InstanceHash  string                 `json:"instanceHash"`
	EventTTLDays  int                    `json:"eventTTLDays"`
	ActiveCRDs    []string               `json:"activeCRDs"`
	ResourceCount map[string]int         `json:"resourceCount"`
	SystemInfo    map[string]interface{} `json:"systemInfo"`
}

// Metrics represents the Prometheus metrics
type Metrics struct {
	resourceEventsTotal *prometheus.CounterVec
	resourceCount       *prometheus.GaugeVec
	uptimeSeconds       prometheus.Gauge
	neo4jConnections    prometheus.Gauge
	registry            *prometheus.Registry
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, k8sClient *kubernetes.Client, neo4jClient *neo4j.Client) *Server {
	return &Server{
		config:      cfg,
		k8sClient:   k8sClient,
		neo4jClient: neo4jClient,
		startTime:   time.Now(),
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	if !s.config.HTTP.Enabled {
		logger.Info("HTTP server is disabled")
		return nil
	}

	// Initialize metrics
	metrics := s.initMetrics()

	// Create mux
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/info", s.handleInfo)
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{}))

	// Create server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.HTTP.Port),
		Handler: mux,
	}

	logger.Info("Starting HTTP server on port %d", s.config.HTTP.Port)

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	// Start metrics collection goroutine
	go s.collectMetrics(ctx, metrics)

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		logger.Info("Stopping HTTP server...")
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleInfo handles the /info endpoint
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get active CRDs
	activeCRDs := s.getActiveCRDs()

	// Get resource counts
	resourceCount := s.getResourceCounts()

	// Get system info
	systemInfo := s.getSystemInfo()

	// Get version info
	versionInfo := version.GetVersionInfo()

	response := InfoResponse{
		Application:   "kubegraph",
		Version:       versionInfo["full"],
		GitCommit:     versionInfo["commit"],
		GitBranch:     versionInfo["branch"],
		StartTime:     s.startTime,
		Uptime:        time.Since(s.startTime).String(),
		ClusterName:   s.config.Kubernetes.ClusterName,
		InstanceHash:  s.config.InstanceHash,
		EventTTLDays:  s.config.EventTTLDays,
		ActiveCRDs:    activeCRDs,
		ResourceCount: resourceCount,
		SystemInfo:    systemInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getActiveCRDs returns a list of active CRDs
func (s *Server) getActiveCRDs() []string {
	if s.k8sClient == nil {
		return []string{}
	}

	// Get all registered handlers (which represent available CRDs)
	handlers := s.k8sClient.GetHandlers()
	activeCRDs := make([]string, 0, len(handlers))

	for _, handler := range handlers {
		gvr := handler.GetGVR()
		crdName := fmt.Sprintf("%s.%s/%s", gvr.Resource, gvr.Group, gvr.Version)
		activeCRDs = append(activeCRDs, crdName)
	}

	return activeCRDs
}

// getResourceCounts returns the count of resources in Neo4j
func (s *Server) getResourceCounts() map[string]int {
	if s.neo4jClient == nil || s.k8sClient == nil {
		return map[string]int{}
	}

	ctx := context.Background()
	resourceCount := make(map[string]int)

	// Dynamically get all resource types from registered handlers
	handlers := s.k8sClient.GetHandlers()
	for kind, handler := range handlers {
		resourceType := handler.GetKind()
		count, err := s.getResourceCount(ctx, resourceType)
		if err != nil {
			logger.Debug("Failed to get count for %s: %v", resourceType, err)
			continue
		}
		resourceCount[resourceType] = count
		// Also allow lookup by handler key for completeness (if different)
		if kind != resourceType {
			resourceCount[kind] = count
		}
	}

	return resourceCount
}

// getResourceCount gets the count of a specific resource type
func (s *Server) getResourceCount(ctx context.Context, resourceType string) (int, error) {
	query := fmt.Sprintf("MATCH (n:%s) WHERE n.clusterName = $clusterName AND n.instanceHash = $instanceHash RETURN count(n) as count", resourceType)

	session := s.neo4jClient.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, map[string]interface{}{
		"clusterName":  s.config.Kubernetes.ClusterName,
		"instanceHash": s.config.InstanceHash,
	})
	if err != nil {
		return 0, err
	}

	record, err := result.Single(ctx)
	if err != nil {
		return 0, err
	}

	count, ok := record.Values[0].(int64)
	if !ok {
		return 0, fmt.Errorf("invalid count type")
	}

	return int(count), nil
}

// getSystemInfo returns system information
func (s *Server) getSystemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"goVersion":     runtime.Version(),
		"goOS":          runtime.GOOS,
		"goArch":        runtime.GOARCH,
		"numCPU":        runtime.NumCPU(),
		"numGoroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc":      m.Alloc,
			"totalAlloc": m.TotalAlloc,
			"sys":        m.Sys,
			"numGC":      m.NumGC,
		},
	}
}

// initMetrics initializes Prometheus metrics
func (s *Server) initMetrics() *Metrics {
	registry := prometheus.NewRegistry()

	metrics := &Metrics{
		resourceEventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kubegraph_resource_events_total",
				Help: "Total number of Kubernetes resource events processed",
			},
			[]string{"resource_type", "event_type", "cluster_name"},
		),
		resourceCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kubegraph_resource_count",
				Help: "Current number of Kubernetes resources in Neo4j",
			},
			[]string{"resource_type", "cluster_name"},
		),
		uptimeSeconds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kubegraph_uptime_seconds",
				Help: "Uptime of the kubegraph service in seconds",
			},
		),
		neo4jConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "kubegraph_neo4j_connections",
				Help: "Number of active Neo4j connections",
			},
		),
		registry: registry,
	}

	// Register metrics
	registry.MustRegister(metrics.resourceEventsTotal)
	registry.MustRegister(metrics.resourceCount)
	registry.MustRegister(metrics.uptimeSeconds)
	registry.MustRegister(metrics.neo4jConnections)

	return metrics
}

// collectMetrics periodically collects and updates metrics
func (s *Server) collectMetrics(ctx context.Context, metrics *Metrics) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update uptime
			metrics.uptimeSeconds.Set(time.Since(s.startTime).Seconds())

			// Update resource counts
			resourceCount := s.getResourceCounts()
			for resourceType, count := range resourceCount {
				metrics.resourceCount.WithLabelValues(resourceType, s.config.Kubernetes.ClusterName).Set(float64(count))
			}

			// Update Neo4j connection status
			if s.neo4jClient != nil {
				// Simple connection check
				session := s.neo4jClient.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeRead})
				_, err := session.Run(ctx, "RETURN 1", nil)
				session.Close(ctx)

				if err == nil {
					metrics.neo4jConnections.Set(1)
				} else {
					metrics.neo4jConnections.Set(0)
				}
			}
		}
	}
}

// IncrementEventCounter increments the event counter for metrics
func (s *Server) IncrementEventCounter(resourceType, eventType string) {
	// This would be called from the Kubernetes handlers
	// For now, we'll implement this when we integrate with the handlers
}
