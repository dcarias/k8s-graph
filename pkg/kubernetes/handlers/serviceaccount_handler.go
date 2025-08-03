package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ServiceAccountHandler struct {
	BaseHandler
	instanceHash string
}

func NewServiceAccountHandler(cfg *config.Config) *ServiceAccountHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "serviceaccounts",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("ServiceAccount", "ServiceAccount")
	return &ServiceAccountHandler{
		BaseHandler:  NewBaseHandler(gvr, "ServiceAccount", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *ServiceAccountHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	sa, err := ConvertToTyped[*corev1.ServiceAccount](obj)
	if err != nil {
		return fmt.Errorf("failed to convert serviceaccount: %w", err)
	}

	properties := map[string]interface{}{
		"name":              sa.Name,
		"uid":               string(sa.UID),
		"namespace":         sa.Namespace,
		"creationTimestamp": sa.CreationTimestamp.String(),
		"labels":            sa.Labels,
		"annotations":       sa.Annotations,
		"secrets":           sa.Secrets,
		"imagePullSecrets":  sa.ImagePullSecrets,
		"automountToken":    sa.AutomountServiceAccountToken,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"ServiceAccount"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert serviceaccount %s: %w", sa.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if sa.OwnerReferences != nil {
		for _, ownerRef := range sa.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"ServiceAccount", "uid", string(sa.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between ServiceAccount %s and %s %s: %v\n", sa.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *ServiceAccountHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	sa, err := ConvertToTyped[*corev1.ServiceAccount](obj)
	if err != nil {
		return fmt.Errorf("failed to convert serviceaccount: %w", err)
	}
	return HandleResourceDelete(ctx, "ServiceAccount", string(sa.UID), neo4jClient)
}
