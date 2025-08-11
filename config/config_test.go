package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	// Test Neo4j configuration
	if cfg.Neo4j.URI != "neo4j://localhost:7687" {
		t.Errorf("Expected Neo4j URI to be 'neo4j://localhost:7687', got '%s'", cfg.Neo4j.URI)
	}
	if cfg.Neo4j.Username != "neo4j" {
		t.Errorf("Expected Neo4j Username to be 'neo4j', got '%s'", cfg.Neo4j.Username)
	}
	if cfg.Neo4j.Password != "password" {
		t.Errorf("Expected Neo4j Password to be 'password', got '%s'", cfg.Neo4j.Password)
	}
	if cfg.Neo4j.MaxConnectionPoolSize != 50 {
		t.Errorf("Expected MaxConnectionPoolSize to be 50, got %d", cfg.Neo4j.MaxConnectionPoolSize)
	}
	if cfg.Neo4j.ConnectionAcquisitionTimeout != 30 {
		t.Errorf("Expected ConnectionAcquisitionTimeout to be 30, got %d", cfg.Neo4j.ConnectionAcquisitionTimeout)
	}
	if cfg.Neo4j.ConnectionLivenessCheckTimeout != 30 {
		t.Errorf("Expected ConnectionLivenessCheckTimeout to be 30, got %d", cfg.Neo4j.ConnectionLivenessCheckTimeout)
	}
	if cfg.Neo4j.MaxConnectionLifetime != 1 {
		t.Errorf("Expected MaxConnectionLifetime to be 1, got %d", cfg.Neo4j.MaxConnectionLifetime)
	}
	if cfg.Neo4j.MaxTransactionRetryTime != 15 {
		t.Errorf("Expected MaxTransactionRetryTime to be 15, got %d", cfg.Neo4j.MaxTransactionRetryTime)
	}

	// Test Kubernetes configuration
	if cfg.Kubernetes.ConfigPath != "" {
		t.Errorf("Expected Kubernetes ConfigPath to be empty, got '%s'", cfg.Kubernetes.ConfigPath)
	}
	if cfg.Kubernetes.ClusterName != "default" {
		t.Errorf("Expected Kubernetes ClusterName to be 'default', got '%s'", cfg.Kubernetes.ClusterName)
	}

	// Test HTTP configuration
	if !cfg.HTTP.Enabled {
		t.Error("Expected HTTP Enabled to be true")
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("Expected HTTP Port to be 8080, got %d", cfg.HTTP.Port)
	}

	// Test other configuration
	if cfg.InstanceHash != "" {
		t.Errorf("Expected InstanceHash to be empty, got '%s'", cfg.InstanceHash)
	}
	if cfg.EventTTLDays != 7 {
		t.Errorf("Expected EventTTLDays to be 7, got %d", cfg.EventTTLDays)
	}
}

func TestConfigStructFields(t *testing.T) {
	cfg := &Config{}

	// Test that all fields can be set
	cfg.Neo4j.URI = "test://uri"
	cfg.Neo4j.Username = "testuser"
	cfg.Neo4j.Password = "testpass"
	cfg.Neo4j.MaxConnectionPoolSize = 100
	cfg.Neo4j.ConnectionAcquisitionTimeout = 60
	cfg.Neo4j.ConnectionLivenessCheckTimeout = 60
	cfg.Neo4j.MaxConnectionLifetime = 2
	cfg.Neo4j.MaxTransactionRetryTime = 30

	cfg.Kubernetes.ConfigPath = "/test/path"
	cfg.Kubernetes.ClusterName = "test-cluster"

	cfg.HTTP.Enabled = false
	cfg.HTTP.Port = 9090

	cfg.InstanceHash = "test-hash"
	cfg.EventTTLDays = 14

	// Verify the values were set correctly
	if cfg.Neo4j.URI != "test://uri" {
		t.Errorf("Failed to set Neo4j URI")
	}
	if cfg.Neo4j.MaxConnectionPoolSize != 100 {
		t.Errorf("Failed to set MaxConnectionPoolSize")
	}
	if cfg.Kubernetes.ClusterName != "test-cluster" {
		t.Errorf("Failed to set ClusterName")
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("Failed to set HTTP Port")
	}
	if cfg.EventTTLDays != 14 {
		t.Errorf("Failed to set EventTTLDays")
	}
}
