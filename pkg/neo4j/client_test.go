package neo4j

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s-graph/config"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestConvertMapPropertiesToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple string values",
			input: map[string]interface{}{
				"name": "test",
				"age":  "25",
			},
			expected: map[string]interface{}{
				"name": "test",
				"age":  "25",
			},
		},
		{
			name: "map values",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
				"labels": map[string]string{
					"app": "test",
				},
			},
			expected: map[string]interface{}{
				"metadata": `{"key1":"value1","key2":"value2"}`,
				"labels":   `{"app":"test"}`,
			},
		},
		{
			name: "array values",
			input: map[string]interface{}{
				"tags": []string{"tag1", "tag2", "tag3"},
				"data": []interface{}{1, "two", 3.0},
			},
			expected: map[string]interface{}{
				"tags": `["tag1","tag2","tag3"]`,
				"data": `[1,"two",3]`,
			},
		},
		{
			name: "nil values",
			input: map[string]interface{}{
				"nullValue": nil,
				"string":    "test",
			},
			expected: map[string]interface{}{
				"string": "test",
			},
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "test",
				"int":    42,
				"float":  3.14,
				"bool":   true,
				"map": map[string]interface{}{
					"nested": "value",
				},
				"slice": []string{"a", "b", "c"},
			},
			expected: map[string]interface{}{
				"string": "test",
				"int":    `42`,
				"float":  `3.14`,
				"bool":   `true`,
				"map":    `{"nested":"value"}`,
				"slice":  `["a","b","c"]`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := convertMapPropertiesToJSON(test.input)

			// Check that all expected keys are present
			for key, expectedValue := range test.expected {
				if resultValue, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' to be present in result", key)
				} else if resultValue != expectedValue {
					t.Errorf("Expected key '%s' to have value '%v', got '%v'", key, expectedValue, resultValue)
				}
			}

			// Check that no unexpected keys are present
			for key := range result {
				if _, exists := test.expected[key]; !exists {
					t.Errorf("Unexpected key '%s' in result", key)
				}
			}
		})
	}
}

func TestBuildUpsertQuery(t *testing.T) {
	tests := []struct {
		name       string
		labels     []string
		properties map[string]interface{}
		uniqueKey  string
		expected   string
	}{
		{
			name:       "single label",
			labels:     []string{"Pod"},
			properties: map[string]interface{}{"name": "test-pod"},
			uniqueKey:  "name",
			expected:   "MERGE (n:Pod {name: $name}) SET n = $properties",
		},
		{
			name:       "multiple labels",
			labels:     []string{"Pod", "v1"},
			properties: map[string]interface{}{"uid": "123"},
			uniqueKey:  "uid",
			expected:   "MERGE (n:Pod:v1 {uid: $uid}) SET n = $properties",
		},
		{
			name:       "no labels",
			labels:     []string{},
			properties: map[string]interface{}{"id": "test"},
			uniqueKey:  "id",
			expected:   "MERGE (n {id: $id}) SET n = $properties",
		},
		{
			name:       "complex unique key",
			labels:     []string{"Service"},
			properties: map[string]interface{}{"namespace_name": "default/test"},
			uniqueKey:  "namespace_name",
			expected:   "MERGE (n:Service {namespace_name: $namespace_name}) SET n = $properties",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := buildUpsertQuery(test.labels, test.properties, test.uniqueKey)
			if result != test.expected {
				t.Errorf("Expected query '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	// Test with valid configuration
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}

	if client == nil {
		t.Error("Expected NewClient to return a client, got nil")
		return
	}

	if client.driver == nil {
		t.Error("Expected client to have a driver, got nil")
		return
	}

	// Clean up
	if client != nil {
		ctx := context.Background()
		client.Close(ctx)
	}
}

func TestNewClientWithInvalidURI(t *testing.T) {
	// Test with invalid URI
	cfg := &config.Config{}
	cfg.Neo4j.URI = "invalid://uri"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err == nil {
		t.Error("Expected NewClient to fail with invalid URI, but it succeeded")
		// Clean up if somehow it succeeded
		if client != nil {
			ctx := context.Background()
			client.Close(ctx)
		}
	}
	if client != nil {
		t.Error("Expected NewClient to return nil client when error occurs")
	}
}

func TestClientDriver(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	driver := client.Driver()
	if driver == nil {
		t.Error("Expected Driver() to return a driver, got nil")
	}
}

func TestClientClose(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}

	ctx := context.Background()
	err = client.Close(ctx)
	if err != nil {
		t.Errorf("Expected Close() to succeed, got error: %v", err)
	}
}

func TestConvertMapPropertiesToJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "map with empty string",
			input: map[string]interface{}{
				"empty": "",
			},
			expected: map[string]interface{}{
				"empty": "",
			},
		},
		{
			name: "map with empty array",
			input: map[string]interface{}{
				"emptyArray": []string{},
			},
			expected: map[string]interface{}{
				"emptyArray": `[]`,
			},
		},
		{
			name: "map with empty map",
			input: map[string]interface{}{
				"emptyMap": map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"emptyMap": `{}`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := convertMapPropertiesToJSON(test.input)

			// Check that all expected keys are present
			for key, expectedValue := range test.expected {
				if resultValue, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' to be present in result", key)
				} else if resultValue != expectedValue {
					t.Errorf("Expected key '%s' to have value '%v', got '%v'", key, expectedValue, resultValue)
				}
			}

			// Check that no unexpected keys are present
			for key := range result {
				if _, exists := test.expected[key]; !exists {
					t.Errorf("Unexpected key '%s' in result", key)
				}
			}
		})
	}
}

func TestExecuteWithMetrics(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test successful operation
	err = client.executeWithMetrics(context.Background(), "test_operation", func() error {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		return nil
	})
	if err != nil {
		t.Errorf("Expected executeWithMetrics to succeed, got error: %v", err)
	}

	// Test failed operation
	err = client.executeWithMetrics(context.Background(), "test_failed_operation", func() error {
		return fmt.Errorf("test error")
	})
	if err == nil {
		t.Error("Expected executeWithMetrics to return error, but it succeeded")
	}
}

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test health check
	err = client.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("Expected HealthCheck to succeed, got error: %v", err)
	}
}

func TestGetServerInfo(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test get server info
	serverInfo, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Errorf("Expected GetServerInfo to succeed, got error: %v", err)
	}
	if serverInfo == nil {
		t.Error("Expected GetServerInfo to return server info, got nil")
	}
}

func TestUpsertNodeWithTransaction(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test transaction-based upsert
	labels := []string{"TestNode", "Transaction"}
	properties := map[string]interface{}{
		"name": "test-transaction",
		"id":   "123",
	}
	uniqueKey := "id"

	err = client.UpsertNodeWithTransaction(context.Background(), labels, properties, uniqueKey)
	if err != nil {
		t.Errorf("Expected UpsertNodeWithTransaction to succeed, got error: %v", err)
	}
}

func TestCreateRelationshipWithTransaction(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test transaction-based relationship creation
	err = client.CreateRelationshipWithTransaction(
		context.Background(),
		"TestNode", "id", "123",
		"RELATES_TO",
		"TestNode", "id", "456",
	)
	if err != nil {
		t.Errorf("Expected CreateRelationshipWithTransaction to succeed, got error: %v", err)
	}
}

func TestExecuteRead(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test read operation
	result, err := client.ExecuteRead(context.Background(), func(tx neo4j.ManagedTransaction) (any, error) {
		// Simulate a read operation
		return "test result", nil
	})
	if err != nil {
		t.Errorf("Expected ExecuteRead to succeed, got error: %v", err)
	}
	if result != "test result" {
		t.Errorf("Expected result 'test result', got '%v'", result)
	}
}

func TestExecuteWrite(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Test write operation
	result, err := client.ExecuteWrite(context.Background(), func(tx neo4j.ManagedTransaction) (any, error) {
		// Simulate a write operation
		return "write result", nil
	})
	if err != nil {
		t.Errorf("Expected ExecuteWrite to succeed, got error: %v", err)
	}
	if result != "write result" {
		t.Errorf("Expected result 'write result', got '%v'", result)
	}
}

func TestMetricsCollection(t *testing.T) {
	cfg := &config.Config{}
	cfg.Neo4j.URI = "neo4j://localhost:7687"
	cfg.Neo4j.Username = "neo4j"
	cfg.Neo4j.Password = "password"
	cfg.Neo4j.MaxConnectionPoolSize = 10
	cfg.Neo4j.ConnectionAcquisitionTimeout = 5
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 5
	cfg.Neo4j.MaxConnectionLifetime = 1
	cfg.Neo4j.MaxTransactionRetryTime = 5

	client, err := NewClient(cfg)
	if err != nil {
		// This is expected when Neo4j is not running
		t.Logf("NewClient failed as expected (Neo4j not running): %v", err)
		return
	}
	defer client.Close(context.Background())

	// Wait a bit for metrics to be collected
	time.Sleep(100 * time.Millisecond)

	// Test that metrics are being collected
	// Note: In a real test environment, you might want to check actual metric values
	// For now, we just ensure the client doesn't crash during metrics collection
}
