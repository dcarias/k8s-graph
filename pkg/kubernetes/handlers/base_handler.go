package handlers

import (
	"context"
	"fmt"

	"k8s-graph/config"
	"k8s-graph/pkg/neo4j"

	driverneo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BaseHandler provides common functionality for resource handlers
type BaseHandler struct {
	gvr         schema.GroupVersionResource
	kind        string
	clusterName string
}

// NewBaseHandler creates a new base handler with common fields
func NewBaseHandler(gvr schema.GroupVersionResource, kind string, cfg *config.Config) BaseHandler {
	return BaseHandler{
		gvr:         gvr,
		kind:        kind,
		clusterName: cfg.Kubernetes.ClusterName,
	}
}

func (h *BaseHandler) GetGVR() schema.GroupVersionResource {
	return h.gvr
}

func (h *BaseHandler) GetKind() string {
	return h.kind
}

func (h *BaseHandler) GetClusterName() string {
	return h.clusterName
}

// HandleResourceDelete is a helper function for deleting resources from Neo4j
func HandleResourceDelete(ctx context.Context, resourceType, uid string, neo4jClient *neo4j.Client) error {
	query := fmt.Sprintf("MATCH (r:%s {uid: $uid}) DETACH DELETE r", resourceType)
	session := neo4jClient.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.Run(ctx, query, map[string]interface{}{"uid": uid})
	return err
}

// ConvertToTyped converts an unstructured object to a typed object
func ConvertToTyped[T any](obj interface{}) (T, error) {
	var zero T

	// Handle the case where the object is already of the desired type
	if typed, ok := obj.(T); ok {
		return typed, nil
	}

	// Handle unstructured objects
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return zero, fmt.Errorf("object is not *unstructured.Unstructured or %T", zero)
	}

	var typedObj T
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &typedObj)
	if err != nil {
		return zero, fmt.Errorf("failed to convert unstructured to typed: %w", err)
	}

	return typedObj, nil
}

var ownerKindToLabel = make(map[string]string)

func RegisterOwnerKind(kind, label string) {
	ownerKindToLabel[kind] = label
}