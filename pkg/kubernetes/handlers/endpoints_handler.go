package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type EndpointsHandler struct {
	BaseHandler
}

func NewEndpointsHandler(cfg *config.Config) *EndpointsHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "endpoints",
	}
	RegisterOwnerKind("Endpoints", "Endpoints")
	return &EndpointsHandler{
		BaseHandler: NewBaseHandler(gvr, "Endpoints", cfg),
	}
}

func (h *EndpointsHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	endpoints, err := ConvertToTyped[*corev1.Endpoints](obj)
	if err != nil {
		return fmt.Errorf("failed to convert endpoints: %w", err)
	}

	// Extract subsets information
	subsets := make([]map[string]interface{}, 0, len(endpoints.Subsets))
	for _, subset := range endpoints.Subsets {
		subsetInfo := map[string]interface{}{
			"addresses": subset.Addresses,
			"ports":     subset.Ports,
		}
		subsets = append(subsets, subsetInfo)
	}

	properties := map[string]interface{}{
		"name":        endpoints.Name,
		"uid":         string(endpoints.UID),
		"namespace":   endpoints.Namespace,
		"subsets":     subsets,
		"labels":      endpoints.Labels,
		"annotations": endpoints.Annotations,
		"clusterName": h.GetClusterName(),
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Endpoints"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert endpoints %s: %w", endpoints.Name, err)
	}

	// Create relationship to the service with the same name
	err = neo4jClient.CreateRelationship(
		ctx,
		"Endpoints", "name", endpoints.Name,
		"PROVIDES_ENDPOINTS_FOR",
		"Service", "name", endpoints.Name,
	)
	if err != nil {
		fmt.Printf("Warning: failed to create PROVIDES_ENDPOINTS_FOR relationship for Endpoints %s: %v\n", endpoints.Name, err)
	}

	return nil
}

func (h *EndpointsHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	endpoints, err := ConvertToTyped[*corev1.Endpoints](obj)
	if err != nil {
		return fmt.Errorf("failed to convert endpoints: %w", err)
	}
	return HandleResourceDelete(ctx, "Endpoints", string(endpoints.UID), neo4jClient)
}
