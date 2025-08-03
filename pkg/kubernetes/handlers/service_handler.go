package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type ServiceHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewServiceHandler(clientset *kubernetes.Clientset, cfg *config.Config) *ServiceHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "services",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("Service", "Service")
	return &ServiceHandler{
		BaseHandler:  NewBaseHandler(gvr, "Service", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *ServiceHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	svc, err := ConvertToTyped[*corev1.Service](obj)
	if err != nil {
		return fmt.Errorf("failed to convert service: %w", err)
	}

	properties := map[string]interface{}{
		"name":              svc.Name,
		"uid":               string(svc.UID),
		"namespace":         svc.Namespace,
		"creationTimestamp": svc.CreationTimestamp.String(),
		"type":              string(svc.Spec.Type),
		"clusterIP":         svc.Spec.ClusterIP,
		"labels":            svc.Labels,
		"annotations":       svc.Annotations,
		"selector":          svc.Spec.Selector,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,
	}

	// Add ports information
	ports := make([]string, 0, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		portInfo := fmt.Sprintf("name=%s;protocol=%s;port=%d;targetPort=%s",
			port.Name,
			port.Protocol,
			port.Port,
			port.TargetPort.String())
		if port.NodePort != 0 {
			portInfo += fmt.Sprintf(";nodePort=%d", port.NodePort)
		}
		ports = append(ports, portInfo)
	}
	properties["ports"] = ports

	// Add external IPs and load balancer info if available
	if len(svc.Spec.ExternalIPs) > 0 {
		properties["externalIPs"] = svc.Spec.ExternalIPs
	}
	if svc.Spec.LoadBalancerIP != "" {
		properties["loadBalancerIP"] = svc.Spec.LoadBalancerIP
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ingress := make([]map[string]string, 0, len(svc.Status.LoadBalancer.Ingress))
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			ingressInfo := make(map[string]string)
			if ing.IP != "" {
				ingressInfo["ip"] = ing.IP
			}
			if ing.Hostname != "" {
				ingressInfo["hostname"] = ing.Hostname
			}
			ingress = append(ingress, ingressInfo)
		}
		properties["loadBalancerIngress"] = ingress
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Service"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert service %s: %w", svc.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if svc.OwnerReferences != nil {
		for _, ownerRef := range svc.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Service", "uid", string(svc.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Service %s and %s %s: %v\n", svc.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with pods based on selector
	if svc.Spec.Selector != nil {
		pods, err := h.clientset.CoreV1().Pods(svc.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: svc.Spec.Selector}),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods for service %s: %w", svc.Name, err)
		}

		for _, pod := range pods.Items {
			if err := neo4jClient.CreateRelationship(ctx, "Service", "uid", string(svc.UID), "SELECTS", "Pod", "uid", string(pod.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between service %s and pod %s: %w", svc.Name, pod.Name, err)
			}
		}
	}

	return nil
}

func (h *ServiceHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	svc, err := ConvertToTyped[*corev1.Service](obj)
	if err != nil {
		return fmt.Errorf("failed to convert service: %w", err)
	}
	return HandleResourceDelete(ctx, "Service", string(svc.UID), neo4jClient)
}
