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

type ReplicaSetHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewReplicaSetHandler(clientset *kubernetes.Clientset, cfg *config.Config) *ReplicaSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "replicasets",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("ReplicaSet", "ReplicaSet")
	return &ReplicaSetHandler{
		BaseHandler:  NewBaseHandler(gvr, "ReplicaSet", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *ReplicaSetHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	rs, err := ConvertToTyped[*appsv1.ReplicaSet](obj)
	if err != nil {
		return fmt.Errorf("failed to convert replicaset: %w", err)
	}

	properties := map[string]interface{}{
		"name":              rs.Name,
		"uid":               string(rs.UID),
		"namespace":         rs.Namespace,
		"creationTimestamp": rs.CreationTimestamp.String(),
		"labels":            rs.Labels,
		"annotations":       rs.Annotations,
		"replicas":          rs.Spec.Replicas,
		"availableReplicas": rs.Status.AvailableReplicas,
		"readyReplicas":     rs.Status.ReadyReplicas,
		"selector":          rs.Spec.Selector.MatchLabels,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"ReplicaSet"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert replicaset %s: %w", rs.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if rs.OwnerReferences != nil {
		for _, ownerRef := range rs.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"ReplicaSet", "uid", string(rs.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between ReplicaSet %s and %s %s: %v\n", rs.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with pods
	if rs.Spec.Selector != nil {
		pods, err := h.clientset.CoreV1().Pods(rs.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(rs.Spec.Selector),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods for replicaset %s: %w", rs.Name, err)
		}

		for _, pod := range pods.Items {
			if err := neo4jClient.CreateRelationship(ctx, "ReplicaSet", "uid", string(rs.UID), "MANAGES", "Pod", "uid", string(pod.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between replicaset %s and pod %s: %w", rs.Name, pod.Name, err)
			}
		}
	}

	return nil
}

func (h *ReplicaSetHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	rs, err := ConvertToTyped[*appsv1.ReplicaSet](obj)
	if err != nil {
		return fmt.Errorf("failed to convert replicaset: %w", err)
	}
	return HandleResourceDelete(ctx, "ReplicaSet", string(rs.UID), neo4jClient)
}
