package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type DeploymentHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewDeploymentHandler(clientset *kubernetes.Clientset, cfg *config.Config) *DeploymentHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("Deployment", "Deployment")
	return &DeploymentHandler{
		BaseHandler:  NewBaseHandler(gvr, "Deployment", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *DeploymentHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	deployment, err := ConvertToTyped[*appsv1.Deployment](obj)
	if err != nil {
		return fmt.Errorf("failed to convert deployment: %w", err)
	}

	properties := map[string]interface{}{
		"name":              deployment.Name,
		"uid":               string(deployment.UID),
		"namespace":         deployment.Namespace,
		"creationTimestamp": deployment.CreationTimestamp.String(),
		"labels":            deployment.Labels,
		"annotations":       deployment.Annotations,
		"replicas":          deployment.Spec.Replicas,
		"strategy":          string(deployment.Spec.Strategy.Type),
		"selector":          deployment.Spec.Selector.MatchLabels,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Deployment"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert deployment %s: %w", deployment.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if deployment.OwnerReferences != nil {
		for _, ownerRef := range deployment.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Deployment", "uid", string(deployment.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Deployment %s and %s %s: %v\n", deployment.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with pods
	if deployment.Spec.Selector != nil {
		pods, err := h.clientset.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods for deployment %s: %w", deployment.Name, err)
		}

		for _, pod := range pods.Items {
			if err := neo4jClient.CreateRelationship(ctx, "Deployment", "uid", string(deployment.UID), "MANAGES", "Pod", "uid", string(pod.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between deployment %s and pod %s: %w", deployment.Name, pod.Name, err)
			}
		}
	}

	return nil
}

func (h *DeploymentHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	deployment, err := ConvertToTyped[*appsv1.Deployment](obj)
	if err != nil {
		return fmt.Errorf("failed to convert deployment: %w", err)
	}
	return HandleResourceDelete(ctx, "Deployment", string(deployment.UID), neo4jClient)
}
