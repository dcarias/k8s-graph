package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type StorageClassHandler struct {
	BaseHandler
	instanceHash string
}

func NewStorageClassHandler(cfg *config.Config) *StorageClassHandler {
	gvr := schema.GroupVersionResource{
		Group:    "storage.k8s.io",
		Version:  "v1",
		Resource: "storageclasses",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("StorageClass", "StorageClass")
	return &StorageClassHandler{
		BaseHandler:  NewBaseHandler(gvr, "StorageClass", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *StorageClassHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	sc, err := ConvertToTyped[*storagev1.StorageClass](obj)
	if err != nil {
		return fmt.Errorf("failed to convert storage class: %w", err)
	}

	properties := map[string]interface{}{
		"name":                 sc.Name,
		"uid":                  string(sc.UID),
		"creationTimestamp":    sc.CreationTimestamp.String(),
		"labels":               sc.Labels,
		"annotations":          sc.Annotations,
		"provisioner":          sc.Provisioner,
		"reclaimPolicy":        string(*sc.ReclaimPolicy),
		"volumeBindingMode":    string(*sc.VolumeBindingMode),
		"allowVolumeExpansion": sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion,
		"clusterName":          h.GetClusterName(),
		"instanceHash":         h.instanceHash,
	}

	// Add parameters if they exist
	if sc.Parameters != nil {
		properties["parameters"] = sc.Parameters
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"StorageClass"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert storage class %s: %w", sc.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if sc.OwnerReferences != nil {
		for _, ownerRef := range sc.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"StorageClass", "uid", string(sc.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between StorageClass %s and %s %s: %v\n", sc.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *StorageClassHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	sc, err := ConvertToTyped[*storagev1.StorageClass](obj)
	if err != nil {
		return fmt.Errorf("failed to convert storage class: %w", err)
	}
	return HandleResourceDelete(ctx, "StorageClass", string(sc.UID), neo4jClient)
}
