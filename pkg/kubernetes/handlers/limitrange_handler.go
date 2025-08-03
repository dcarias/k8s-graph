package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type LimitRangeHandler struct {
	BaseHandler
	instanceHash string
}

func NewLimitRangeHandler(cfg *config.Config) *LimitRangeHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "limitranges",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("LimitRange", "LimitRange")
	return &LimitRangeHandler{
		BaseHandler:  NewBaseHandler(gvr, "LimitRange", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *LimitRangeHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	lr, err := ConvertToTyped[*corev1.LimitRange](obj)
	if err != nil {
		return fmt.Errorf("failed to convert limitrange: %w", err)
	}

	properties := map[string]interface{}{
		"name":              lr.Name,
		"uid":               string(lr.UID),
		"namespace":         lr.Namespace,
		"creationTimestamp": lr.CreationTimestamp.String(),
		"labels":            lr.Labels,
		"annotations":       lr.Annotations,
		"spec":              lr.Spec,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"LimitRange"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert limitrange %s: %w", lr.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if lr.OwnerReferences != nil {
		for _, ownerRef := range lr.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"LimitRange", "uid", string(lr.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between LimitRange %s and %s %s: %v\n", lr.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *LimitRangeHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	lr, err := ConvertToTyped[*corev1.LimitRange](obj)
	if err != nil {
		return fmt.Errorf("failed to convert limitrange: %w", err)
	}
	return HandleResourceDelete(ctx, "LimitRange", string(lr.UID), neo4jClient)
}
