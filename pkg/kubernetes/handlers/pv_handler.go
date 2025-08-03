package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type PVHandler struct {
	BaseHandler
	instanceHash string
}

func NewPVHandler(cfg *config.Config) *PVHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "persistentvolumes",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("PersistentVolume", "PersistentVolume")
	return &PVHandler{
		BaseHandler:  NewBaseHandler(gvr, "PersistentVolume", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *PVHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pv, err := ConvertToTyped[*corev1.PersistentVolume](obj)
	if err != nil {
		return fmt.Errorf("failed to convert persistent volume: %w", err)
	}

	properties := map[string]interface{}{
		"name":              pv.Name,
		"uid":               string(pv.UID),
		"creationTimestamp": pv.CreationTimestamp.String(),
		"labels":            pv.Labels,
		"annotations":       pv.Annotations,
		"capacity":          pv.Spec.Capacity.Storage().String(),
		"accessModes":       convertAccessModes(pv.Spec.AccessModes),
		"reclaimPolicy":     string(pv.Spec.PersistentVolumeReclaimPolicy),
		"status":            string(pv.Status.Phase),
		"storageClass":      pv.Spec.StorageClassName,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"PersistentVolume"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert persistent volume %s: %w", pv.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if pv.OwnerReferences != nil {
		for _, ownerRef := range pv.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"PersistentVolume", "uid", string(pv.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between PersistentVolume %s and %s %s: %v\n", pv.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationship with PVC if bound
	if pv.Spec.ClaimRef != nil {
		if err := neo4jClient.CreateRelationship(ctx, "PersistentVolume", "uid", string(pv.UID), "BOUND_TO", "PersistentVolumeClaim", "uid", string(pv.Spec.ClaimRef.UID)); err != nil {
			return fmt.Errorf("failed to create relationship between PV %s and PVC %s: %w", pv.Name, pv.Spec.ClaimRef.Name, err)
		}
	}

	return nil
}

func (h *PVHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pv, err := ConvertToTyped[*corev1.PersistentVolume](obj)
	if err != nil {
		return fmt.Errorf("failed to convert persistent volume: %w", err)
	}
	return HandleResourceDelete(ctx, "PersistentVolume", string(pv.UID), neo4jClient)
}

func convertAccessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	result := make([]string, len(modes))
	for i, mode := range modes {
		result[i] = string(mode)
	}
	return result
}
