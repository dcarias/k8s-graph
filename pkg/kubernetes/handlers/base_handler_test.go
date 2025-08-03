package handlers

import (
	"testing"

	"kubegraph/config"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewBaseHandler(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kubernetes.ClusterName = "test-cluster"

	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	kind := "Deployment"

	handler := NewBaseHandler(gvr, kind, cfg)

	if handler.gvr != gvr {
		t.Errorf("Expected GVR to be %v, got %v", gvr, handler.gvr)
	}
	if handler.kind != kind {
		t.Errorf("Expected kind to be %s, got %s", kind, handler.kind)
	}
	if handler.clusterName != "test-cluster" {
		t.Errorf("Expected cluster name to be 'test-cluster', got %s", handler.clusterName)
	}
}

func TestBaseHandlerMethods(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kubernetes.ClusterName = "test-cluster"

	gvr := schema.GroupVersionResource{
		Group:    "core",
		Version:  "v1",
		Resource: "pods",
	}
	kind := "Pod"

	handler := NewBaseHandler(gvr, kind, cfg)

	// Test GetGVR
	resultGVR := handler.GetGVR()
	if resultGVR != gvr {
		t.Errorf("Expected GetGVR() to return %v, got %v", gvr, resultGVR)
	}

	// Test GetKind
	resultKind := handler.GetKind()
	if resultKind != kind {
		t.Errorf("Expected GetKind() to return %s, got %s", kind, resultKind)
	}

	// Test GetClusterName
	resultClusterName := handler.GetClusterName()
	if resultClusterName != "test-cluster" {
		t.Errorf("Expected GetClusterName() to return 'test-cluster', got %s", resultClusterName)
	}
}

func TestRegisterOwnerKind(t *testing.T) {
	// Test registering owner kinds
	RegisterOwnerKind("Deployment", "app.kubernetes.io/name")
	RegisterOwnerKind("ReplicaSet", "app.kubernetes.io/name")

	// Check that the mappings were registered
	if ownerKindToLabel["Deployment"] != "app.kubernetes.io/name" {
		t.Errorf("Expected Deployment to be mapped to 'app.kubernetes.io/name', got %s", ownerKindToLabel["Deployment"])
	}
	if ownerKindToLabel["ReplicaSet"] != "app.kubernetes.io/name" {
		t.Errorf("Expected ReplicaSet to be mapped to 'app.kubernetes.io/name', got %s", ownerKindToLabel["ReplicaSet"])
	}

	// Test overwriting existing mapping
	RegisterOwnerKind("Deployment", "new-label")
	if ownerKindToLabel["Deployment"] != "new-label" {
		t.Errorf("Expected Deployment to be updated to 'new-label', got %s", ownerKindToLabel["Deployment"])
	}
}

func TestConvertToTyped(t *testing.T) {
	// Test converting unstructured to typed
	unstructuredObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "nginx",
						"image": "nginx:latest",
					},
				},
			},
		},
	}

	// Test converting to unstructured (should work)
	result, err := ConvertToTyped[*unstructured.Unstructured](unstructuredObj)
	if err != nil {
		t.Errorf("Expected ConvertToTyped to succeed for unstructured, got error: %v", err)
	}
	if result != unstructuredObj {
		t.Error("Expected ConvertToTyped to return the same unstructured object")
	}

	// Test converting to a different type (should succeed if the field exists)
	type TestStruct struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
	}
	typed, err := ConvertToTyped[TestStruct](unstructuredObj)
	if err != nil {
		t.Errorf("Expected ConvertToTyped to succeed for partial struct, got error: %v", err)
	}
	if typed.Metadata.Name != "test-pod" {
		t.Errorf("Expected Metadata.Name to be 'test-pod', got '%s'", typed.Metadata.Name)
	}
}

func TestConvertToTypedWithAlreadyTyped(t *testing.T) {
	// Test with object that's already of the desired type
	unstructuredObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
		},
	}

	result, err := ConvertToTyped[*unstructured.Unstructured](unstructuredObj)
	if err != nil {
		t.Errorf("Expected ConvertToTyped to succeed for already typed object, got error: %v", err)
	}
	if result != unstructuredObj {
		t.Error("Expected ConvertToTyped to return the same object when already typed")
	}
}

func TestConvertToTypedWithInvalidObject(t *testing.T) {
	// Test with object that's neither unstructured nor the desired type
	invalidObj := "not-an-object"

	type TestStruct struct {
		Name string `json:"name"`
	}
	_, err := ConvertToTyped[TestStruct](invalidObj)
	if err == nil {
		t.Error("Expected ConvertToTyped to fail for invalid object type, but it succeeded")
	}
}

func TestBaseHandlerWithDifferentConfigs(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		gvr         schema.GroupVersionResource
		kind        string
	}{
		{
			name:        "default cluster",
			clusterName: "default",
			gvr: schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			kind: "Deployment",
		},
		{
			name:        "production cluster",
			clusterName: "production",
			gvr: schema.GroupVersionResource{
				Group:    "core",
				Version:  "v1",
				Resource: "services",
			},
			kind: "Service",
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			gvr: schema.GroupVersionResource{
				Group:    "batch",
				Version:  "v1",
				Resource: "jobs",
			},
			kind: "Job",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Kubernetes.ClusterName = test.clusterName

			handler := NewBaseHandler(test.gvr, test.kind, cfg)

			if handler.GetGVR() != test.gvr {
				t.Errorf("Expected GVR to be %v, got %v", test.gvr, handler.GetGVR())
			}
			if handler.GetKind() != test.kind {
				t.Errorf("Expected kind to be %s, got %s", test.kind, handler.GetKind())
			}
			if handler.GetClusterName() != test.clusterName {
				t.Errorf("Expected cluster name to be %s, got %s", test.clusterName, handler.GetClusterName())
			}
		})
	}
}
