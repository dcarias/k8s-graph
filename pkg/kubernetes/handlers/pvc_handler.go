// At the top of pkg/kubernetes/handlers/pvc_handler.go
package handlers

import (
	"context"
	"fmt"
	"strings"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewPVCHandler(cfg *config.Config) *PVCHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "persistentvolumeclaims",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("PersistentVolumeClaim", "PersistentVolumeClaim")
	return &PVCHandler{
		BaseHandler:  NewBaseHandler(gvr, "PersistentVolumeClaim", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

type PVCHandler struct {
	BaseHandler
	instanceHash string
}

func (h *PVCHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pvc, err := ConvertToTyped[*corev1.PersistentVolumeClaim](obj)
	if err != nil {
		return fmt.Errorf("failed to convert persistent volume claim: %w", err)
	}

	properties := map[string]interface{}{
		"name":              pvc.Name,
		"uid":               string(pvc.UID),
		"namespace":         pvc.Namespace,
		"creationTimestamp": pvc.CreationTimestamp.String(),
		"labels":            pvc.Labels,
		"annotations":       pvc.Annotations,
		"storageClass":      pvc.Spec.StorageClassName,
		"accessModes":       convertAccessModes(pvc.Spec.AccessModes),
		"volumeName":        pvc.Spec.VolumeName,
		"status":            string(pvc.Status.Phase),
		"capacity":          pvc.Status.Capacity.Storage().String(),
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"PersistentVolumeClaim"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert persistent volume claim %s: %w", pvc.Name, err)
	}

	// Create relationship with PV if bound
	if pvc.Spec.VolumeName != "" {
		if err := neo4jClient.CreateRelationship(ctx, "PersistentVolumeClaim", "uid", string(pvc.UID), "BOUND_TO", "PersistentVolume", "name", pvc.Spec.VolumeName); err != nil {
			return fmt.Errorf("failed to create relationship between PVC %s and PV %s: %w", pvc.Name, pvc.Spec.VolumeName, err)
		}
	}

	// Create relationships based on owner references for all supported types
	if pvc.OwnerReferences != nil {
		for _, ownerRef := range pvc.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"PersistentVolumeClaim", "uid", string(pvc.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between PVC %s and %s %s: %v\n", pvc.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships to StatefulSets that use this PVC
	// This is based on the PVC name pattern and StatefulSet naming convention
	// PVC names follow pattern: data-p-<clusterid or singleinstanceid>-<podIndex>-<volumeIndex>
	// Example: data-p-e1229598-12f1-0001-0
	// StatefulSet names follow pattern: p-<clusterid or singleinstanceid>-<podIndex>
	// Example: p-e1229598-12f1-0001
	if strings.HasPrefix(pvc.Name, "data-p-") {
		// Remove "data-" prefix to get: p-<clusterid or singleinstanceid>-<podIndex>-<volumeIndex>
		nameWithoutPrefix := strings.TrimPrefix(pvc.Name, "data-")
		parts := strings.Split(nameWithoutPrefix, "-")
		if len(parts) >= 4 {
			// parts[0] = "p"
			// parts[1] = first part of clusterid (e.g., "e1229598")
			// parts[2] = second part of clusterid or podIndex (e.g., "12f1" or "0001")
			// parts[3] = podIndex or volumeIndex (e.g., "0001" or "0")
			// parts[4] = volumeIndex (e.g., "0") - if exists

			var statefulSetName string
			if len(parts) == 4 {
				// Format: p-<clusterid>-<podIndex>-<volumeIndex>
				// Example: p-e1229598-0001-0
				statefulSetName = fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
			} else if len(parts) >= 5 {
				// Format: p-<clusterid>-<clusterid-suffix>-<podIndex>-<volumeIndex>
				// Example: p-e1229598-12f1-0001-0
				statefulSetName = fmt.Sprintf("%s-%s-%s-%s", parts[0], parts[1], parts[2], parts[3])
			}

			// Try to create relationship with StatefulSet
			if statefulSetName != "" {
				if err := neo4jClient.CreateRelationship(ctx, "PersistentVolumeClaim", "uid", string(pvc.UID), "USED_BY", "StatefulSet", "name", statefulSetName); err != nil {
					fmt.Printf("Warning: failed to create relationship between PVC %s and StatefulSet %s: %v\n", pvc.Name, statefulSetName, err)
				}
			}
		}
	}

	// Create relationships based on labels if available (fallback for when owner references are not set)
	if pvc.Labels != nil {
		// Check for dbid label
		if dbid, exists := pvc.Labels["dbid"]; exists {
			if err := neo4jClient.CreateRelationship(ctx, "PersistentVolumeClaim", "uid", string(pvc.UID), "OWNED_BY", "Neo4jSingleInstance", "dbid", dbid); err != nil {
				fmt.Printf("Warning: failed to create relationship between PVC %s and Neo4jSingleInstance %s (via label): %v\n", pvc.Name, dbid, err)
			}
		}

		// Check for cluster-related labels
		if clusterId, exists := pvc.Labels["clusterId"]; exists {
			if err := neo4jClient.CreateRelationship(ctx, "PersistentVolumeClaim", "uid", string(pvc.UID), "OWNED_BY", "Neo4jCluster", "clusterId", clusterId); err != nil {
				fmt.Printf("Warning: failed to create relationship between PVC %s and Neo4jCluster %s (via label): %v\n", pvc.Name, clusterId, err)
			}
		}
	}

	return nil
}

func (h *PVCHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pvc, err := ConvertToTyped[*corev1.PersistentVolumeClaim](obj)
	if err != nil {
		return fmt.Errorf("failed to convert persistent volume claim: %w", err)
	}
	return HandleResourceDelete(ctx, "PersistentVolumeClaim", string(pvc.UID), neo4jClient)
}
