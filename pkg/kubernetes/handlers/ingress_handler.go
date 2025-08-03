package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type IngressHandler struct {
	BaseHandler
}

func NewIngressHandler(cfg *config.Config) *IngressHandler {
	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	}
	RegisterOwnerKind("Ingress", "Ingress")
	return &IngressHandler{
		BaseHandler: NewBaseHandler(gvr, "Ingress", cfg),
	}
}

func (h *IngressHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	ingress, err := ConvertToTyped[*networkingv1.Ingress](obj)
	if err != nil {
		return fmt.Errorf("failed to convert ingress: %w", err)
	}

	// Extract ingress rules
	rules := make([]map[string]interface{}, 0, len(ingress.Spec.Rules))
	for _, rule := range ingress.Spec.Rules {
		ruleInfo := map[string]interface{}{
			"host": rule.Host,
		}
		if rule.HTTP != nil {
			paths := make([]map[string]interface{}, 0, len(rule.HTTP.Paths))
			for _, path := range rule.HTTP.Paths {
				pathInfo := map[string]interface{}{
					"path":     path.Path,
					"pathType": string(*path.PathType),
				}
				if path.Backend.Service != nil {
					pathInfo["serviceName"] = path.Backend.Service.Name
					pathInfo["servicePort"] = path.Backend.Service.Port.Number
				}
				paths = append(paths, pathInfo)
			}
			ruleInfo["paths"] = paths
		}
		rules = append(rules, ruleInfo)
	}

	// Extract TLS configuration
	tls := make([]map[string]interface{}, 0, len(ingress.Spec.TLS))
	for _, tlsConfig := range ingress.Spec.TLS {
		tlsInfo := map[string]interface{}{
			"secretName": tlsConfig.SecretName,
		}
		if len(tlsConfig.Hosts) > 0 {
			tlsInfo["hosts"] = tlsConfig.Hosts
		}
		tls = append(tls, tlsInfo)
	}

	// Extract load balancer status
	var loadBalancerStatus map[string]interface{}
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		loadBalancerStatus = map[string]interface{}{
			"ingress": ingress.Status.LoadBalancer.Ingress,
		}
	}

	properties := map[string]interface{}{
		"name":               ingress.Name,
		"uid":                string(ingress.UID),
		"namespace":          ingress.Namespace,
		"ingressClassName":   ingress.Spec.IngressClassName,
		"rules":              rules,
		"tls":                tls,
		"loadBalancerStatus": loadBalancerStatus,
		"labels":             ingress.Labels,
		"annotations":        ingress.Annotations,
		"clusterName":        h.GetClusterName(),
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Ingress"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert ingress %s: %w", ingress.Name, err)
	}

	// Create relationships to referenced services
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					err := neo4jClient.CreateRelationship(
						ctx,
						"Ingress", "uid", string(ingress.UID),
						"ROUTES_TO",
						"Service", "name", path.Backend.Service.Name,
					)
					if err != nil {
						fmt.Printf("Warning: failed to create ROUTES_TO relationship for Ingress %s: %v\n", ingress.Name, err)
					}
				}
			}
		}
	}

	return nil
}

func (h *IngressHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	ingress, err := ConvertToTyped[*networkingv1.Ingress](obj)
	if err != nil {
		return fmt.Errorf("failed to convert ingress: %w", err)
	}
	return HandleResourceDelete(ctx, "Ingress", string(ingress.UID), neo4jClient)
}
