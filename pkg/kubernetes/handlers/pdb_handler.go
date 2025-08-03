package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type PodDisruptionBudgetHandler struct {
	BaseHandler
	instanceHash string
}

func NewPodDisruptionBudgetHandler(cfg *config.Config) *PodDisruptionBudgetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "policy",
		Version:  "v1",
		Resource: "poddisruptionbudgets",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("PodDisruptionBudget", "PodDisruptionBudget")
	return &PodDisruptionBudgetHandler{
		BaseHandler:  NewBaseHandler(gvr, "PodDisruptionBudget", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *PodDisruptionBudgetHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pdb, err := ConvertToTyped[*policyv1.PodDisruptionBudget](obj)
	if err != nil {
		return fmt.Errorf("failed to convert poddisruptionbudget: %w", err)
	}

	properties := map[string]interface{}{
		"name":                       pdb.Name,
		"uid":                        string(pdb.UID),
		"namespace":                  pdb.Namespace,
		"creationTimestamp":          pdb.CreationTimestamp.String(),
		"labels":                     pdb.Labels,
		"annotations":                pdb.Annotations,
		"minAvailable":               pdb.Spec.MinAvailable,
		"maxUnavailable":             pdb.Spec.MaxUnavailable,
		"selector":                   pdb.Spec.Selector,
		"unhealthyPodEvictionPolicy": pdb.Spec.UnhealthyPodEvictionPolicy,
		"currentHealthy":             pdb.Status.CurrentHealthy,
		"desiredHealthy":             pdb.Status.DesiredHealthy,
		"expectedPods":               pdb.Status.ExpectedPods,
		"disruptionsAllowed":         pdb.Status.DisruptionsAllowed,
		"conditions":                 pdb.Status.Conditions,
		"clusterName":                h.GetClusterName(),
		"instanceHash":               h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"PodDisruptionBudget"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert poddisruptionbudget %s: %w", pdb.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if pdb.OwnerReferences != nil {
		for _, ownerRef := range pdb.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"PodDisruptionBudget", "uid", string(pdb.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between PodDisruptionBudget %s and %s %s: %v\n", pdb.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *PodDisruptionBudgetHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pdb, err := ConvertToTyped[*policyv1.PodDisruptionBudget](obj)
	if err != nil {
		return fmt.Errorf("failed to convert poddisruptionbudget: %w", err)
	}
	return HandleResourceDelete(ctx, "PodDisruptionBudget", string(pdb.UID), neo4jClient)
}
