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

type StatefulSetHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewStatefulSetHandler(clientset *kubernetes.Clientset, cfg *config.Config) *StatefulSetHandler {
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("StatefulSet", "StatefulSet")
	return &StatefulSetHandler{
		BaseHandler:  NewBaseHandler(gvr, "StatefulSet", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *StatefulSetHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	sts, err := ConvertToTyped[*appsv1.StatefulSet](obj)
	if err != nil {
		return fmt.Errorf("failed to convert statefulset: %w", err)
	}

	properties := map[string]interface{}{
		"name":              sts.Name,
		"uid":               string(sts.UID),
		"namespace":         sts.Namespace,
		"creationTimestamp": sts.CreationTimestamp.String(),
		"labels":            sts.Labels,
		"annotations":       sts.Annotations,
		"replicas":          sts.Spec.Replicas,
		"serviceName":       sts.Spec.ServiceName,
		"selector":          sts.Spec.Selector.MatchLabels,
		"updateStrategy":    string(sts.Spec.UpdateStrategy.Type),
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"StatefulSet"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert statefulset %s: %w", sts.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if sts.OwnerReferences != nil {
		for _, ownerRef := range sts.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"StatefulSet", "uid", string(sts.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between StatefulSet %s and %s %s: %v\n", sts.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with pods
	if sts.Spec.Selector != nil {
		pods, err := h.clientset.CoreV1().Pods(sts.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(sts.Spec.Selector),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods for statefulset %s: %w", sts.Name, err)
		}

		for _, pod := range pods.Items {
			if err := neo4jClient.CreateRelationship(ctx, "StatefulSet", "uid", string(sts.UID), "MANAGES", "Pod", "uid", string(pod.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between statefulset %s and pod %s: %w", sts.Name, pod.Name, err)
			}
		}
	}

	// Create relationship with service if it exists
	if sts.Spec.ServiceName != "" {
		svc, err := h.clientset.CoreV1().Services(sts.Namespace).Get(ctx, sts.Spec.ServiceName, metav1.GetOptions{})
		if err == nil {
			if err := neo4jClient.CreateRelationship(ctx, "StatefulSet", "uid", string(sts.UID), "USES", "Service", "uid", string(svc.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between statefulset %s and service %s: %w", sts.Name, svc.Name, err)
			}
		}
	}

	return nil
}

func (h *StatefulSetHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	sts, err := ConvertToTyped[*appsv1.StatefulSet](obj)
	if err != nil {
		return fmt.Errorf("failed to convert statefulset: %w", err)
	}
	return HandleResourceDelete(ctx, "StatefulSet", string(sts.UID), neo4jClient)
}
