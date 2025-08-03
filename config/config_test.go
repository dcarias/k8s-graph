package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	// Test Neo4j defaults
	if cfg.Neo4j.URI != "neo4j://localhost:7687" {
		t.Errorf("Expected Neo4j URI to be 'neo4j://localhost:7687', got '%s'", cfg.Neo4j.URI)
	}
	if cfg.Neo4j.Username != "neo4j" {
		t.Errorf("Expected Neo4j username to be 'neo4j', got '%s'", cfg.Neo4j.Username)
	}
	if cfg.Neo4j.Password != "password" {
		t.Errorf("Expected Neo4j password to be 'password', got '%s'", cfg.Neo4j.Password)
	}

	// Test Kubernetes defaults
	if cfg.Kubernetes.ConfigPath != "" {
		t.Errorf("Expected Kubernetes config path to be empty, got '%s'", cfg.Kubernetes.ConfigPath)
	}
	if cfg.Kubernetes.ClusterName != "default" {
		t.Errorf("Expected Kubernetes cluster name to be 'default', got '%s'", cfg.Kubernetes.ClusterName)
	}

	// Test HTTP defaults
	if !cfg.HTTP.Enabled {
		t.Error("Expected HTTP to be enabled by default")
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("Expected HTTP port to be 8080, got %d", cfg.HTTP.Port)
	}

	// Test other defaults
	if cfg.InstanceHash != "" {
		t.Errorf("Expected instance hash to be empty, got '%s'", cfg.InstanceHash)
	}
	if cfg.EventTTLDays != 7 {
		t.Errorf("Expected event TTL days to be 7, got %d", cfg.EventTTLDays)
	}
}

func TestConfigStructFields(t *testing.T) {
	cfg := NewConfig()

	// Test that we can set and get values
	cfg.Neo4j.URI = "neo4j://test:7687"
	cfg.Neo4j.Username = "testuser"
	cfg.Neo4j.Password = "testpass"
	cfg.Kubernetes.ClusterName = "test-cluster"
	cfg.HTTP.Port = 9090
	cfg.HTTP.Enabled = false
	cfg.InstanceHash = "test-hash"
	cfg.EventTTLDays = 30

	if cfg.Neo4j.URI != "neo4j://test:7687" {
		t.Errorf("Expected Neo4j URI to be 'neo4j://test:7687', got '%s'", cfg.Neo4j.URI)
	}
	if cfg.Neo4j.Username != "testuser" {
		t.Errorf("Expected Neo4j username to be 'testuser', got '%s'", cfg.Neo4j.Username)
	}
	if cfg.Neo4j.Password != "testpass" {
		t.Errorf("Expected Neo4j password to be 'testpass', got '%s'", cfg.Neo4j.Password)
	}
	if cfg.Kubernetes.ClusterName != "test-cluster" {
		t.Errorf("Expected Kubernetes cluster name to be 'test-cluster', got '%s'", cfg.Kubernetes.ClusterName)
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("Expected HTTP port to be 9090, got %d", cfg.HTTP.Port)
	}
	if cfg.HTTP.Enabled {
		t.Error("Expected HTTP to be disabled")
	}
	if cfg.InstanceHash != "test-hash" {
		t.Errorf("Expected instance hash to be 'test-hash', got '%s'", cfg.InstanceHash)
	}
	if cfg.EventTTLDays != 30 {
		t.Errorf("Expected event TTL days to be 30, got %d", cfg.EventTTLDays)
	}
}
