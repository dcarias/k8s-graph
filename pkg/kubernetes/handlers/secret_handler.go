package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type SecretHandler struct {
	BaseHandler
	instanceHash string
}

func NewSecretHandler(cfg *config.Config) *SecretHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "secrets",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("Secret", "Secret")
	return &SecretHandler{
		BaseHandler:  NewBaseHandler(gvr, "Secret", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *SecretHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	secret, err := ConvertToTyped[*corev1.Secret](obj)
	if err != nil {
		return fmt.Errorf("failed to convert secret: %w", err)
	}

	properties := map[string]interface{}{
		"name":              secret.Name,
		"uid":               string(secret.UID),
		"namespace":         secret.Namespace,
		"creationTimestamp": secret.CreationTimestamp.String(),
		"labels":            secret.Labels,
		"annotations":       secret.Annotations,
		"type":              string(secret.Type),
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Secret"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert secret %s: %w", secret.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if secret.OwnerReferences != nil {
		for _, ownerRef := range secret.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Secret", "uid", string(secret.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Secret %s and %s %s: %v\n", secret.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *SecretHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	secret, err := ConvertToTyped[*corev1.Secret](obj)
	if err != nil {
		return fmt.Errorf("failed to convert secret: %w", err)
	}
	return HandleResourceDelete(ctx, "Secret", string(secret.UID), neo4jClient)
}
