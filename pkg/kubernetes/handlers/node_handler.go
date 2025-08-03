package handlers

import (
	"context"
	"fmt"
	"strings"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NodeHandler struct {
	BaseHandler
	instanceHash string
}

func NewNodeHandler(cfg *config.Config) *NodeHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "nodes",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("Node", "Node")
	return &NodeHandler{
		BaseHandler:  NewBaseHandler(gvr, "Node", cfg),
		instanceHash: cfg.InstanceHash,
	}
}

func (h *NodeHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	node, err := ConvertToTyped[*corev1.Node](obj)
	if err != nil {
		return fmt.Errorf("failed to convert node: %w", err)
	}

	// Extract capacity
	capacity := node.Status.Capacity
	allocatable := node.Status.Allocatable

	// Get node conditions
	conditions := make(map[string]string)
	for _, condition := range node.Status.Conditions {
		conditions[string(condition.Type)] = string(condition.Status)
	}

	// Get node addresses
	addresses := make(map[string]string)
	for _, addr := range node.Status.Addresses {
		addresses[string(addr.Type)] = addr.Address
	}

	// Extract taints
	taints := make([]string, 0)
	for _, taint := range node.Spec.Taints {
		taints = append(taints, fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect))
	}

	properties := map[string]interface{}{
		"name":              node.Name,
		"uid":               string(node.UID),
		"creationTimestamp": node.CreationTimestamp.String(),
		"labels":            node.Labels,
		"annotations":       node.Annotations,
		"clusterName":       h.GetClusterName(),
		"instanceHash":      h.instanceHash,

		// Capacity
		"capacityCPU":    capacity.Cpu().String(),
		"capacityMemory": capacity.Memory().String(),
		"capacityPods":   capacity.Pods().String(),

		// Allocatable
		"allocatableCPU":    allocatable.Cpu().String(),
		"allocatableMemory": allocatable.Memory().String(),
		"allocatablePods":   allocatable.Pods().String(),

		// Node Info
		"architecture":     node.Status.NodeInfo.Architecture,
		"containerRuntime": node.Status.NodeInfo.ContainerRuntimeVersion,
		"kernelVersion":    node.Status.NodeInfo.KernelVersion,
		"kubeletVersion":   node.Status.NodeInfo.KubeletVersion,
		"osImage":          node.Status.NodeInfo.OSImage,
		"operatingSystem":  node.Status.NodeInfo.OperatingSystem,

		// Status
		"conditions":    conditions,
		"addresses":     addresses,
		"taints":        taints,
		"unschedulable": node.Spec.Unschedulable,
		"phase":         string(node.Status.Phase),
	}

	// Add ephemeral storage if available
	if ephemeralStorage, ok := capacity["ephemeral-storage"]; ok {
		properties["capacityEphemeralStorage"] = ephemeralStorage.String()
	}
	if ephemeralStorage, ok := allocatable["ephemeral-storage"]; ok {
		properties["allocatableEphemeralStorage"] = ephemeralStorage.String()
	}

	// Add extended resources if any
	for key, value := range capacity {
		if strings.HasPrefix(string(key), "nvidia.com/") || strings.HasPrefix(string(key), "amd.com/") {
			properties["capacity"+string(key)] = value.String()
		}
	}
	for key, value := range allocatable {
		if strings.HasPrefix(string(key), "nvidia.com/") || strings.HasPrefix(string(key), "amd.com/") {
			properties["allocatable"+string(key)] = value.String()
		}
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Node"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert node %s: %w", node.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if node.OwnerReferences != nil {
		for _, ownerRef := range node.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Node", "uid", string(node.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Node %s and %s %s: %v\n", node.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	return nil
}

func (h *NodeHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	node, err := ConvertToTyped[*corev1.Node](obj)
	if err != nil {
		return fmt.Errorf("failed to convert node: %w", err)
	}
	return HandleResourceDelete(ctx, "Node", string(node.UID), neo4jClient)
}
