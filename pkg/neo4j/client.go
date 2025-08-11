package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s-graph/config"
	"k8s-graph/pkg/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the Neo4j client
var (
	neo4jOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "neo4j_operations_total",
			Help: "Total number of Neo4j operations",
		},
		[]string{"operation", "status"},
	)

	neo4jOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "neo4j_operation_duration_seconds",
			Help:    "Duration of Neo4j operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	neo4jActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "neo4j_active_sessions",
			Help: "Number of currently active Neo4j sessions",
		},
	)

	neo4jConnectionPoolSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "neo4j_connection_pool_size",
			Help: "Current size of the Neo4j connection pool",
		},
	)

	neo4jConnectionPoolInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "neo4j_connection_pool_in_use",
			Help: "Number of connections currently in use in the Neo4j connection pool",
		},
	)

	neo4jConnectionPoolIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "neo4j_connection_pool_idle",
			Help: "Number of idle connections in the Neo4j connection pool",
		},
	)
)

// Client represents a Neo4j client with connection pooling and metrics
type Client struct {
	driver neo4j.DriverWithContext
	config *config.Config
	mu     sync.RWMutex
}

// NewClient creates a new Neo4j client with optimized connection pooling
func NewClient(cfg *config.Config) (*Client, error) {
	// Configure connection pooling
	driverConfig := neo4j.Config{
		MaxConnectionPoolSize:          cfg.Neo4j.MaxConnectionPoolSize,
		ConnectionAcquisitionTimeout:   time.Duration(cfg.Neo4j.ConnectionAcquisitionTimeout) * time.Second,
		ConnectionLivenessCheckTimeout: time.Duration(cfg.Neo4j.ConnectionLivenessCheckTimeout) * time.Second,
		MaxConnectionLifetime:          time.Duration(cfg.Neo4j.MaxConnectionLifetime) * time.Hour,
		MaxTransactionRetryTime:        time.Duration(cfg.Neo4j.MaxTransactionRetryTime) * time.Second,
	}

	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4j.URI,
		neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""),
		func(config *neo4j.Config) {
			*config = driverConfig
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify neo4j connectivity: %w", err)
	}

	client := &Client{
		driver: driver,
		config: cfg,
	}

	// Start metrics collection goroutine
	go client.collectMetrics()

	return client, nil
}

// collectMetrics periodically collects connection pool metrics
func (c *Client) collectMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.updateConnectionPoolMetrics()
		}
	}
}

// updateConnectionPoolMetrics updates connection pool related metrics
func (c *Client) updateConnectionPoolMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get connection pool statistics
	stats, err := c.driver.GetServerInfo(ctx)
	if err == nil && stats != nil {
		// Note: The Neo4j Go driver doesn't expose detailed connection pool metrics
		// We'll use reasonable defaults and update when available
		neo4jConnectionPoolSize.Set(float64(c.config.Neo4j.MaxConnectionPoolSize))
		neo4jConnectionPoolInUse.Set(0)                                            // Will be updated when we have better metrics
		neo4jConnectionPoolIdle.Set(float64(c.config.Neo4j.MaxConnectionPoolSize)) // Will be updated when we have better metrics
	}
}

// Close closes the Neo4j driver and all connections
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// convertMapPropertiesToJSON converts map properties to JSON strings
func convertMapPropertiesToJSON(properties map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range properties {
		switch val := v.(type) {
		case map[string]interface{}, map[string]string, map[string][]string:
			// Convert any map type to JSON string
			if jsonStr, err := json.Marshal(val); err == nil {
				result[k] = string(jsonStr)
			} else {
				result[k] = "{}" // fallback to empty JSON object if marshaling fails
			}
		case []interface{}, []string:
			// Convert arrays to JSON string
			if jsonStr, err := json.Marshal(val); err == nil {
				result[k] = string(jsonStr)
			} else {
				result[k] = "[]" // fallback to empty JSON array if marshaling fails
			}
		case string:
			result[k] = val
		case nil:
			// Do not add this key at all!
			continue
		default:
			// Try to marshal unknown types, fall back to string representation
			if jsonStr, err := json.Marshal(val); err == nil {
				result[k] = string(jsonStr)
			} else {
				result[k] = fmt.Sprintf("%v", val)
			}
		}
	}
	return result
}

// executeWithMetrics executes a Neo4j operation with metrics collection
func (c *Client) executeWithMetrics(ctx context.Context, operation string, fn func() error) error {
	start := time.Now()
	neo4jActiveSessions.Inc()
	defer func() {
		neo4jActiveSessions.Dec()
		duration := time.Since(start).Seconds()
		neo4jOperationDuration.WithLabelValues(operation).Observe(duration)
	}()

	err := fn()
	if err != nil {
		neo4jOperationsTotal.WithLabelValues(operation, "error").Inc()
		return err
	}

	neo4jOperationsTotal.WithLabelValues(operation, "success").Inc()
	return nil
}

// UpsertNode creates or updates a node with the given labels and properties
func (c *Client) UpsertNode(ctx context.Context, labels []string, properties map[string]interface{}, uniqueKey string) error {
	return c.executeWithMetrics(ctx, "upsert_node", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		// Convert map properties to JSON strings
		convertedProperties := convertMapPropertiesToJSON(properties)

		query := buildUpsertQuery(labels, convertedProperties, uniqueKey)
		params := map[string]interface{}{
			uniqueKey:    properties[uniqueKey], // Use original value for unique key
			"properties": convertedProperties,
		}

		_, err := session.Run(ctx, query, params)
		return err
	})
}

// UpsertNodeWithTransaction creates or updates a node within a transaction
func (c *Client) UpsertNodeWithTransaction(ctx context.Context, labels []string, properties map[string]interface{}, uniqueKey string) error {
	return c.executeWithMetrics(ctx, "upsert_node_transaction", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			convertedProperties := convertMapPropertiesToJSON(properties)
			query := buildUpsertQuery(labels, convertedProperties, uniqueKey)
			params := map[string]interface{}{
				uniqueKey:    properties[uniqueKey],
				"properties": convertedProperties,
			}

			_, err := tx.Run(ctx, query, params)
			return nil, err
		})

		return err
	})
}

func buildUpsertQuery(labels []string, properties map[string]interface{}, uniqueKey string) string {
	labelStr := ""
	for _, label := range labels {
		labelStr += ":" + label
	}
	return fmt.Sprintf("MERGE (n%s {%s: $%s}) SET n = $properties", labelStr, uniqueKey, uniqueKey)
}

// CreateRelationship creates a relationship between two nodes
func (c *Client) CreateRelationship(ctx context.Context, fromNodeLabel, fromNodeKey, fromNodeValue, relationshipType, toNodeLabel, toNodeKey, toNodeValue string) error {
	return c.executeWithMetrics(ctx, "create_relationship", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		query := fmt.Sprintf(`
			MATCH (from:%s {%s: $fromValue})
			MATCH (to:%s {%s: $toValue})
			MERGE (from)-[r:%s]->(to)
			RETURN r`, fromNodeLabel, fromNodeKey, toNodeLabel, toNodeKey, relationshipType)

		params := map[string]interface{}{
			"fromValue": fromNodeValue,
			"toValue":   toNodeValue,
		}

		_, err := session.Run(ctx, query, params)
		return err
	})
}

// CreateRelationshipWithTransaction creates a relationship within a transaction
func (c *Client) CreateRelationshipWithTransaction(ctx context.Context, fromNodeLabel, fromNodeKey, fromNodeValue, relationshipType, toNodeLabel, toNodeKey, toNodeValue string) error {
	return c.executeWithMetrics(ctx, "create_relationship_transaction", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := fmt.Sprintf(`
				MATCH (from:%s {%s: $fromValue})
				MATCH (to:%s {%s: $toValue})
				MERGE (from)-[r:%s]->(to)
				RETURN r`, fromNodeLabel, fromNodeKey, toNodeLabel, toNodeKey, relationshipType)

			params := map[string]interface{}{
				"fromValue": fromNodeValue,
				"toValue":   toNodeValue,
			}

			_, err := tx.Run(ctx, query, params)
			return nil, err
		})

		return err
	})
}

// ExecuteRead executes a read operation with proper session management
func (c *Client) ExecuteRead(ctx context.Context, fn func(neo4j.ManagedTransaction) (any, error)) (any, error) {
	var result any
	err := c.executeWithMetrics(ctx, "execute_read", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeRead,
		})
		defer session.Close(ctx)

		var execErr error
		result, execErr = session.ExecuteRead(ctx, fn)
		return execErr
	})
	return result, err
}

// ExecuteWrite executes a write operation with proper session management
func (c *Client) ExecuteWrite(ctx context.Context, fn func(neo4j.ManagedTransaction) (any, error)) (any, error) {
	var result any
	err := c.executeWithMetrics(ctx, "execute_write", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		var execErr error
		result, execErr = session.ExecuteWrite(ctx, fn)
		return execErr
	})
	return result, err
}

// Driver returns the underlying Neo4j driver
func (c *Client) Driver() neo4j.DriverWithContext {
	return c.driver
}

// DeleteOldClustersByName deletes clusters with the same name but different instance hashes
func (c *Client) DeleteOldClustersByName(ctx context.Context, clusterName, currentInstanceHash string) error {
	return c.executeWithMetrics(ctx, "delete_old_clusters", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		// Delete clusters with the same name but different instance hash
		query := `
			MATCH (c:Neo4jCluster {clusterName: $clusterName})
			WHERE c.instanceHash IS NOT NULL AND c.instanceHash <> $currentInstanceHash
			DETACH DELETE c`

		params := map[string]interface{}{
			"clusterName":         clusterName,
			"currentInstanceHash": currentInstanceHash,
		}

		_, err := session.Run(ctx, query, params)
		return err
	})
}

// DeleteOldResourcesByClusterName deletes resources with the same cluster name but different instance hashes
func (c *Client) DeleteOldResourcesByClusterName(ctx context.Context, resourceType, clusterName, currentInstanceHash string) error {
	return c.executeWithMetrics(ctx, "delete_old_resources", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		// Delete resources with the same cluster name but different instance hash
		query := fmt.Sprintf(`
			MATCH (r:%s {clusterName: $clusterName})
			WHERE r.instanceHash IS NOT NULL AND r.instanceHash <> $currentInstanceHash
			DETACH DELETE r`, resourceType)

		params := map[string]interface{}{
			"clusterName":         clusterName,
			"currentInstanceHash": currentInstanceHash,
		}

		_, err := session.Run(ctx, query, params)
		return err
	})
}

// CleanupDuplicateClusters removes all nodes with the same cluster name but different instance hashes
// Events are excluded from this cleanup as they should be preserved across runs
func (c *Client) CleanupDuplicateClusters(ctx context.Context, clusterName, currentInstanceHash string) error {
	return c.executeWithMetrics(ctx, "cleanup_duplicate_clusters", func() error {
		session := c.driver.NewSession(ctx, neo4j.SessionConfig{
			AccessMode: neo4j.AccessModeWrite,
		})
		defer session.Close(ctx)

		// Clean up all resource types that have clusterName and instanceHash properties
		// Events are excluded as they should be preserved across runs
		query := `
			MATCH (n)
			WHERE n.clusterName = $clusterName 
			AND n.instanceHash IS NOT NULL 
			AND n.instanceHash <> $currentInstanceHash
			AND NOT n:Event
			DETACH DELETE n`

		params := map[string]interface{}{
			"clusterName":         clusterName,
			"currentInstanceHash": currentInstanceHash,
		}

		result, err := session.Run(ctx, query, params)
		if err != nil {
			return err
		}

		// Get the summary to see how many nodes were deleted
		summary, err := result.Consume(ctx)
		if err != nil {
			return err
		}

		if summary.Counters().NodesDeleted() > 0 {
			logger.Info("Cleaned up %d duplicate nodes for cluster %s with old instance hash (Events excluded)", summary.Counters().NodesDeleted(), clusterName)
		}

		return nil
	})
}

// HealthCheck performs a health check on the Neo4j connection
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.executeWithMetrics(ctx, "health_check", func() error {
		return c.driver.VerifyConnectivity(ctx)
	})
}

// GetServerInfo returns information about the Neo4j server
func (c *Client) GetServerInfo(ctx context.Context) (neo4j.ServerInfo, error) {
	var serverInfo neo4j.ServerInfo
	err := c.executeWithMetrics(ctx, "get_server_info", func() error {
		info, infoErr := c.driver.GetServerInfo(ctx)
		if infoErr != nil {
			return infoErr
		}
		if info == nil {
			return fmt.Errorf("failed to get server info")
		}
		serverInfo = info
		return nil
	})
	return serverInfo, err
}
