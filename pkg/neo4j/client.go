package neo4j

import (
	"context"
	"encoding/json"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/logger"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Client struct {
	driver neo4j.DriverWithContext
}

func NewClient(cfg *config.Config) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(cfg.Neo4j.URI, neo4j.BasicAuth(cfg.Neo4j.Username, cfg.Neo4j.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	return &Client{
		driver: driver,
	}, nil
}

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

// UpsertNode creates or updates a node with the given labels and properties
func (c *Client) UpsertNode(ctx context.Context, labels []string, properties map[string]interface{}, uniqueKey string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
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
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
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
}

// Driver returns the underlying Neo4j driver
func (c *Client) Driver() neo4j.DriverWithContext {
	return c.driver
}

// DeleteOldClustersByName deletes clusters with the same name but different instance hashes
func (c *Client) DeleteOldClustersByName(ctx context.Context, clusterName, currentInstanceHash string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
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
}

// DeleteOldResourcesByClusterName deletes resources with the same cluster name but different instance hashes
func (c *Client) DeleteOldResourcesByClusterName(ctx context.Context, resourceType, clusterName, currentInstanceHash string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
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
}

// CleanupDuplicateClusters removes all nodes with the same cluster name but different instance hashes
// Events are excluded from this cleanup as they should be preserved across runs
func (c *Client) CleanupDuplicateClusters(ctx context.Context, clusterName, currentInstanceHash string) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
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
}
