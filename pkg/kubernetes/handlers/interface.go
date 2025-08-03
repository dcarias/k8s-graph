package handlers

import (
	"context"

	"k8s-graph/pkg/neo4j"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceHandler defines the interface for handling Kubernetes resources
type ResourceHandler interface {
	// GetGVR returns the GroupVersionResource for this handler
	GetGVR() schema.GroupVersionResource
	// GetKind returns the kind of resource this handler manages
	GetKind() string
	// HandleCreate handles creation/update events
	HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error
	// HandleDelete handles deletion events
	HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error
}