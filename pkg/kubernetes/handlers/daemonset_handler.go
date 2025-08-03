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

type DaemonSetHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewDaemonSetHandler(clientset *kubernetes.Clientset, cfg *config.Config) *DaemonSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("DaemonSet", "DaemonSet")
	return &DaemonSetHandler{
		BaseHandler:  NewBaseHandler(gvr, "DaemonSet", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *DaemonSetHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	ds, err := ConvertToTyped[*appsv1.DaemonSet](obj)
	if err != nil {
		return fmt.Errorf("failed to convert daemonset: %w", err)
	}

	properties := map[string]interface{}{
		"name":              ds.Name,
		"uid":               string(ds.UID),
		"namespace":         ds.Namespace,
		"creationTimestamp": ds.CreationTimestamp.String(),
		"labels":            ds.Labels,
		"annotations":       ds.Annotations,
		"selector":          ds.Spec.Selector.MatchLabels,
		"updateStrategy":    string(ds.Spec.UpdateStrategy.Type),
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"DaemonSet"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert daemonset %s: %w", ds.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if ds.OwnerReferences != nil {
		for _, ownerRef := range ds.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"DaemonSet", "uid", string(ds.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between DaemonSet %s and %s %s: %v\n", ds.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with pods
	if ds.Spec.Selector != nil {
		pods, err := h.clientset.CoreV1().Pods(ds.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(ds.Spec.Selector),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods for daemonset %s: %w", ds.Name, err)
		}

		for _, pod := range pods.Items {
			if err := neo4jClient.CreateRelationship(ctx, "DaemonSet", "uid", string(ds.UID), "MANAGES", "Pod", "uid", string(pod.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between daemonset %s and pod %s: %w", ds.Name, pod.Name, err)
			}
		}
	}

	return nil
}

func (h *DaemonSetHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	ds, err := ConvertToTyped[*appsv1.DaemonSet](obj)
	if err != nil {
		return fmt.Errorf("failed to convert daemonset: %w", err)
	}
	return HandleResourceDelete(ctx, "DaemonSet", string(ds.UID), neo4jClient)
}
