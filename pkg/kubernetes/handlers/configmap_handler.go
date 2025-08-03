package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ConfigMapHandler struct {
	BaseHandler
	instanceHash string
}

func NewConfigMapHandler(cfg *config.Config) *ConfigMapHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "configmaps",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("ConfigMap", "ConfigMap")
	return &ConfigMapHandler{
		BaseHandler:  NewBaseHandler(gvr, "ConfigMap", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *ConfigMapHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	cm, err := ConvertToTyped[*corev1.ConfigMap](obj)
	if err != nil {
		return fmt.Errorf("failed to convert configmap: %w", err)
	}

	properties := map[string]interface{}{
		"name":              cm.Name,
		"uid":               string(cm.UID),
		"namespace":         cm.Namespace,
		"creationTimestamp": cm.CreationTimestamp.String(),
		"labels":            cm.Labels,
		"annotations":       cm.Annotations,
		"data":              cm.Data,
		"binaryData":        cm.BinaryData,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"ConfigMap"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert configmap %s: %w", cm.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if cm.OwnerReferences != nil {
		for _, ownerRef := range cm.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"ConfigMap", "uid", string(cm.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between ConfigMap %s and %s %s: %v\n", cm.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *ConfigMapHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	cm, err := ConvertToTyped[*corev1.ConfigMap](obj)
	if err != nil {
		return fmt.Errorf("failed to convert configmap: %w", err)
	}
	return HandleResourceDelete(ctx, "ConfigMap", string(cm.UID), neo4jClient)
}
