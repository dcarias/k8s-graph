package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type HorizontalPodAutoscalerHandler struct {
	BaseHandler
	instanceHash string
}

func NewHorizontalPodAutoscalerHandler(cfg *config.Config) *HorizontalPodAutoscalerHandler {
	gvr := schema.GroupVersionResource{
		Group:    "autoscaling",
		Version:  "v2",
		Resource: "horizontalpodautoscalers",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("HorizontalPodAutoscaler", "HorizontalPodAutoscaler")
	return &HorizontalPodAutoscalerHandler{
		BaseHandler:  NewBaseHandler(gvr, "HorizontalPodAutoscaler", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *HorizontalPodAutoscalerHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	hpa, err := ConvertToTyped[*autoscalingv2.HorizontalPodAutoscaler](obj)
	if err != nil {
		return fmt.Errorf("failed to convert horizontalpodautoscaler: %w", err)
	}

	properties := map[string]interface{}{
		"name":              hpa.Name,
		"uid":               string(hpa.UID),
		"namespace":         hpa.Namespace,
		"creationTimestamp": hpa.CreationTimestamp.String(),
		"labels":            hpa.Labels,
		"annotations":       hpa.Annotations,
		"minReplicas":       hpa.Spec.MinReplicas,
		"maxReplicas":       hpa.Spec.MaxReplicas,
		"scaleTargetRef":    hpa.Spec.ScaleTargetRef,
		"metrics":           hpa.Spec.Metrics,
		"behavior":          hpa.Spec.Behavior,
		"currentReplicas":   hpa.Status.CurrentReplicas,
		"desiredReplicas":   hpa.Status.DesiredReplicas,
		"currentMetrics":    hpa.Status.CurrentMetrics,
		"conditions":        hpa.Status.Conditions,
		"lastScaleTime":     hpa.Status.LastScaleTime,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"HorizontalPodAutoscaler"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert horizontalpodautoscaler %s: %w", hpa.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if hpa.OwnerReferences != nil {
		for _, ownerRef := range hpa.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"HorizontalPodAutoscaler", "uid", string(hpa.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between HorizontalPodAutoscaler %s and %s %s: %v\n", hpa.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationship to the target resource (Deployment, StatefulSet, etc.)
	if hpa.Spec.ScaleTargetRef.Name != "" {
		targetKind := hpa.Spec.ScaleTargetRef.Kind
		if label, ok := ownerKindToLabel[targetKind]; ok {
			err := neo4jClient.CreateRelationship(
				ctx,
				"HorizontalPodAutoscaler", "uid", string(hpa.UID),
				"SCALES",
				label, "name", hpa.Spec.ScaleTargetRef.Name,
			)
			if err != nil {
				fmt.Printf("Warning: failed to create SCALES relationship between HorizontalPodAutoscaler %s and %s %s: %v\n", hpa.Name, targetKind, hpa.Spec.ScaleTargetRef.Name, err)
			}
		}
	}

	return nil
}

func (h *HorizontalPodAutoscalerHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	hpa, err := ConvertToTyped[*autoscalingv2.HorizontalPodAutoscaler](obj)
	if err != nil {
		return fmt.Errorf("failed to convert horizontalpodautoscaler: %w", err)
	}
	return HandleResourceDelete(ctx, "HorizontalPodAutoscaler", string(hpa.UID), neo4jClient)
}
