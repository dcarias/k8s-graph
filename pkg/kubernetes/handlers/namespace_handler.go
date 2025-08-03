package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NamespaceHandler struct {
	BaseHandler
	instanceHash string
}

func NewNamespaceHandler(cfg *config.Config) *NamespaceHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "namespaces",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("Namespace", "Namespace")
	return &NamespaceHandler{
		BaseHandler:  NewBaseHandler(gvr, "Namespace", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *NamespaceHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	ns, err := ConvertToTyped[*corev1.Namespace](obj)
	if err != nil {
		return fmt.Errorf("failed to convert namespace: %w", err)
	}

	properties := map[string]interface{}{
		"name":              ns.Name,
		"uid":               string(ns.UID),
		"creationTimestamp": ns.CreationTimestamp.String(),
		"labels":            ns.Labels,
		"annotations":       ns.Annotations,
		"status":            string(ns.Status.Phase),
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Namespace"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert namespace %s: %w", ns.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if ns.OwnerReferences != nil {
		for _, ownerRef := range ns.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Namespace", "uid", string(ns.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Namespace %s and %s %s: %v\n", ns.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *NamespaceHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	ns, err := ConvertToTyped[*corev1.Namespace](obj)
	if err != nil {
		return fmt.Errorf("failed to convert namespace: %w", err)
	}
	return HandleResourceDelete(ctx, "Namespace", string(ns.UID), neo4jClient)
}
