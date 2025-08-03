package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NetworkPolicyHandler struct {
	BaseHandler
}

func NewNetworkPolicyHandler(cfg *config.Config) *NetworkPolicyHandler {
	gvr := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "networkpolicies",
	}
	RegisterOwnerKind("NetworkPolicy", "NetworkPolicy")
	return &NetworkPolicyHandler{
		BaseHandler: NewBaseHandler(gvr, "NetworkPolicy", cfg),
	}
}

func (h *NetworkPolicyHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	networkPolicy, err := ConvertToTyped[*networkingv1.NetworkPolicy](obj)
	if err != nil {
		return fmt.Errorf("failed to convert networkpolicy: %w", err)
	}

	// Extract policy types
	policyTypes := make([]string, 0, len(networkPolicy.Spec.PolicyTypes))
	for _, policyType := range networkPolicy.Spec.PolicyTypes {
		policyTypes = append(policyTypes, string(policyType))
	}

	// Extract ingress rules
	ingress := make([]map[string]interface{}, 0, len(networkPolicy.Spec.Ingress))
	for _, rule := range networkPolicy.Spec.Ingress {
		ruleInfo := map[string]interface{}{}
		if len(rule.Ports) > 0 {
			ports := make([]map[string]interface{}, 0, len(rule.Ports))
			for _, port := range rule.Ports {
				portInfo := map[string]interface{}{
					"protocol": string(*port.Protocol),
				}
				if port.Port != nil {
					portInfo["port"] = port.Port.String()
				}
				if port.EndPort != nil {
					portInfo["endPort"] = *port.EndPort
				}
				ports = append(ports, portInfo)
			}
			ruleInfo["ports"] = ports
		}
		if len(rule.From) > 0 {
			from := make([]map[string]interface{}, 0, len(rule.From))
			for _, fromRule := range rule.From {
				fromInfo := map[string]interface{}{}
				if fromRule.NamespaceSelector != nil {
					fromInfo["namespaceSelector"] = fromRule.NamespaceSelector.MatchLabels
				}
				if fromRule.PodSelector != nil {
					fromInfo["podSelector"] = fromRule.PodSelector.MatchLabels
				}
				if fromRule.IPBlock != nil {
					fromInfo["ipBlock"] = map[string]interface{}{
						"cidr":   fromRule.IPBlock.CIDR,
						"except": fromRule.IPBlock.Except,
					}
				}
				from = append(from, fromInfo)
			}
			ruleInfo["from"] = from
		}
		ingress = append(ingress, ruleInfo)
	}

	// Extract egress rules
	egress := make([]map[string]interface{}, 0, len(networkPolicy.Spec.Egress))
	for _, rule := range networkPolicy.Spec.Egress {
		ruleInfo := map[string]interface{}{}
		if len(rule.Ports) > 0 {
			ports := make([]map[string]interface{}, 0, len(rule.Ports))
			for _, port := range rule.Ports {
				portInfo := map[string]interface{}{
					"protocol": string(*port.Protocol),
				}
				if port.Port != nil {
					portInfo["port"] = port.Port.String()
				}
				if port.EndPort != nil {
					portInfo["endPort"] = *port.EndPort
				}
				ports = append(ports, portInfo)
			}
			ruleInfo["ports"] = ports
		}
		if len(rule.To) > 0 {
			to := make([]map[string]interface{}, 0, len(rule.To))
			for _, toRule := range rule.To {
				toInfo := map[string]interface{}{}
				if toRule.NamespaceSelector != nil {
					toInfo["namespaceSelector"] = toRule.NamespaceSelector.MatchLabels
				}
				if toRule.PodSelector != nil {
					toInfo["podSelector"] = toRule.PodSelector.MatchLabels
				}
				if toRule.IPBlock != nil {
					toInfo["ipBlock"] = map[string]interface{}{
						"cidr":   toRule.IPBlock.CIDR,
						"except": toRule.IPBlock.Except,
					}
				}
				to = append(to, toInfo)
			}
			ruleInfo["to"] = to
		}
		egress = append(egress, ruleInfo)
	}

	properties := map[string]interface{}{
		"name":        networkPolicy.Name,
		"uid":         string(networkPolicy.UID),
		"namespace":   networkPolicy.Namespace,
		"policyTypes": policyTypes,
		"ingress":     ingress,
		"egress":      egress,
		"labels":      networkPolicy.Labels,
		"annotations": networkPolicy.Annotations,
		"clusterName": h.GetClusterName(),
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"NetworkPolicy"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert networkpolicy %s: %w", networkPolicy.Name, err)
	}

	return nil
}

func (h *NetworkPolicyHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	networkPolicy, err := ConvertToTyped[*networkingv1.NetworkPolicy](obj)
	if err != nil {
		return fmt.Errorf("failed to convert networkpolicy: %w", err)
	}
	return HandleResourceDelete(ctx, "NetworkPolicy", string(networkPolicy.UID), neo4jClient)
}
