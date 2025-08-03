package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type VerticalPodAutoscalerHandler struct {
	BaseHandler
	instanceHash string
}

func NewVerticalPodAutoscalerHandler(cfg *config.Config) *VerticalPodAutoscalerHandler {
	gvr := schema.GroupVersionResource{
		Group:    "autoscaling.k8s.io",
		Version:  "v1",
		Resource: "verticalpodautoscalers",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("VerticalPodAutoscaler", "VerticalPodAutoscaler")
	return &VerticalPodAutoscalerHandler{
		BaseHandler:  NewBaseHandler(gvr, "VerticalPodAutoscaler", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *VerticalPodAutoscalerHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	// Since VPA is not in the standard Kubernetes API, we'll work with unstructured objects
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("object is not *unstructured.Unstructured")
	}

	// Extract basic properties
	name := unstructuredObj.GetName()
	uid := string(unstructuredObj.GetUID())
	namespace := unstructuredObj.GetNamespace()
	creationTimestamp := unstructuredObj.GetCreationTimestamp().String()
	labels := unstructuredObj.GetLabels()
	annotations := unstructuredObj.GetAnnotations()

	// Extract VPA-specific properties
	spec, _ := unstructuredObj.Object["spec"].(map[string]interface{})
	status, _ := unstructuredObj.Object["status"].(map[string]interface{})

	properties := map[string]interface{}{
		"name":              name,
		"uid":               uid,
		"namespace":         namespace,
		"creationTimestamp": creationTimestamp,
		"labels":            labels,
		"annotations":       annotations,
		"spec":              spec,
		"status":            status,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"VerticalPodAutoscaler"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert verticalpodautoscaler %s: %w", name, err)
	}

	// Create relationships based on owner references for all supported types
	ownerReferences := unstructuredObj.GetOwnerReferences()
	if ownerReferences != nil {
		for _, ownerRef := range ownerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"VerticalPodAutoscaler", "uid", uid,
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between VerticalPodAutoscaler %s and %s %s: %v\n", name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationship to the target resource (Deployment, StatefulSet, etc.)
	if spec != nil {
		if targetRef, ok := spec["targetRef"].(map[string]interface{}); ok {
			if targetName, ok := targetRef["name"].(string); ok && targetName != "" {
				if targetKind, ok := targetRef["kind"].(string); ok {
					if label, ok := ownerKindToLabel[targetKind]; ok {
						err := neo4jClient.CreateRelationship(
							ctx,
							"VerticalPodAutoscaler", "uid", uid,
							"SCALES",
							label, "name", targetName,
						)
						if err != nil {
							fmt.Printf("Warning: failed to create SCALES relationship between VerticalPodAutoscaler %s and %s %s: %v\n", name, targetKind, targetName, err)
						}
					}
				}
			}
		}
	}

	return nil
}

func (h *VerticalPodAutoscalerHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("object is not *unstructured.Unstructured")
	}
	return HandleResourceDelete(ctx, "VerticalPodAutoscaler", string(unstructuredObj.GetUID()), neo4jClient)
}
