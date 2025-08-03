package handlers

import (
	"testing"

	"kubegraph/config"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewNamespaceHandler(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kubernetes.ClusterName = "test-cluster"
	cfg.InstanceHash = "test-hash"

	handler := NewNamespaceHandler(cfg)

	// Test that the handler was created correctly
	expectedGVR := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "namespaces",
	}
	if handler.GetGVR() != expectedGVR {
		t.Errorf("Expected GVR to be %v, got %v", expectedGVR, handler.GetGVR())
	}

	if handler.GetKind() != "Namespace" {
		t.Errorf("Expected kind to be 'Namespace', got %s", handler.GetKind())
	}

	if handler.GetClusterName() != "test-cluster" {
		t.Errorf("Expected cluster name to be 'test-cluster', got %s", handler.GetClusterName())
	}

	if handler.instanceHash != "test-hash" {
		t.Errorf("Expected instance hash to be 'test-hash', got %s", handler.instanceHash)
	}

	// Test that the owner kind was registered
	if ownerKindToLabel["Namespace"] != "Namespace" {
		t.Errorf("Expected Namespace to be registered with label 'Namespace', got %s", ownerKindToLabel["Namespace"])
	}
}

func TestNamespaceHandlerWithDifferentConfigs(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		instanceHash string
	}{
		{
			name:         "default cluster",
			clusterName:  "default",
			instanceHash: "hash-1",
		},
		{
			name:         "production cluster",
			clusterName:  "production",
			instanceHash: "hash-2",
		},
		{
			name:         "empty cluster name",
			clusterName:  "",
			instanceHash: "hash-3",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Kubernetes.ClusterName = test.clusterName
			cfg.InstanceHash = test.instanceHash

			handler := NewNamespaceHandler(cfg)

			if handler.GetGVR().Version != "v1" {
				t.Errorf("Expected GVR version to be 'v1', got %s", handler.GetGVR().Version)
			}
			if handler.GetGVR().Resource != "namespaces" {
				t.Errorf("Expected GVR resource to be 'namespaces', got %s", handler.GetGVR().Resource)
			}
			if handler.GetKind() != "Namespace" {
				t.Errorf("Expected kind to be 'Namespace', got %s", handler.GetKind())
			}
			if handler.GetClusterName() != test.clusterName {
				t.Errorf("Expected cluster name to be %s, got %s", test.clusterName, handler.GetClusterName())
			}
			if handler.instanceHash != test.instanceHash {
				t.Errorf("Expected instance hash to be %s, got %s", test.instanceHash, handler.instanceHash)
			}
		})
	}
}

func TestNamespaceHandlerImplementsResourceHandler(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kubernetes.ClusterName = "test-cluster"
	cfg.InstanceHash = "test-hash"

	handler := NewNamespaceHandler(cfg)

	// Test that the handler implements the ResourceHandler interface
	var _ ResourceHandler = handler

	// Test that all required methods exist and return expected types
	gvr := handler.GetGVR()
	if gvr.Version != "v1" {
		t.Errorf("Expected GVR version to be 'v1', got %s", gvr.Version)
	}

	kind := handler.GetKind()
	if kind != "Namespace" {
		t.Errorf("Expected kind to be 'Namespace', got %s", kind)
	}

	clusterName := handler.GetClusterName()
	if clusterName != "test-cluster" {
		t.Errorf("Expected cluster name to be 'test-cluster', got %s", clusterName)
	}
}

func TestNamespaceHandlerOwnerKindRegistration(t *testing.T) {
	// Clear any existing registrations for this test
	delete(ownerKindToLabel, "Namespace")

	cfg := &config.Config{}
	cfg.Kubernetes.ClusterName = "test-cluster"
	cfg.InstanceHash = "test-hash"

	// Create handler which should register the owner kind
	handler := NewNamespaceHandler(cfg)

	// Verify registration
	if ownerKindToLabel["Namespace"] != "Namespace" {
		t.Errorf("Expected Namespace to be registered with label 'Namespace', got %s", ownerKindToLabel["Namespace"])
	}

	// Test that the handler has the correct instance hash
	if handler.instanceHash != "test-hash" {
		t.Errorf("Expected instance hash to be 'test-hash', got %s", handler.instanceHash)
	}
}
