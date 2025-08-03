package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type PodHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	clusterName  string
	instanceHash string
}

func NewPodHandler(clientset *kubernetes.Clientset, cfg *config.Config) *PodHandler {
	// Register this handler's kind for owner references
	RegisterOwnerKind("Pod", "Pod")
	return &PodHandler{
		BaseHandler: BaseHandler{
			gvr: schema.GroupVersionResource{
				Version:  "v1",
				Resource: "pods",
			},
			kind: "Pod",
		},
		clientset:    clientset,
		clusterName:  cfg.Kubernetes.ClusterName,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *PodHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pod, err := ConvertToTyped[*corev1.Pod](obj)
	if err != nil {
		return fmt.Errorf("failed to convert pod: %w", err)
	}

	if pod == nil {
		return fmt.Errorf("pod is nil after conversion")
	}

	// Get pod conditions
	conditions := make(map[string]string)
	if pod.Status.Conditions != nil {
		for _, condition := range pod.Status.Conditions {
			conditions[string(condition.Type)] = string(condition.Status)
		}
	}

	// Calculate total resource requests and limits
	var totalRequests, totalLimits corev1.ResourceList
	totalRequests = make(corev1.ResourceList)
	totalLimits = make(corev1.ResourceList)

	// Container statuses and security contexts
	containerStatuses := make([]string, 0)
	containerSecurityContexts := make([]map[string]interface{}, 0)
	if pod.Spec.Containers != nil {
		for _, container := range pod.Spec.Containers {
			// Add container resources to totals
			if container.Resources.Requests != nil {
				for resourceName, quantity := range container.Resources.Requests {
					if currentQuantity, ok := totalRequests[resourceName]; ok {
						currentQuantity.Add(quantity)
						totalRequests[resourceName] = currentQuantity
					} else {
						totalRequests[resourceName] = quantity.DeepCopy()
					}
				}
			}
			if container.Resources.Limits != nil {
				for resourceName, quantity := range container.Resources.Limits {
					if currentQuantity, ok := totalLimits[resourceName]; ok {
						currentQuantity.Add(quantity)
						totalLimits[resourceName] = currentQuantity
					} else {
						totalLimits[resourceName] = quantity.DeepCopy()
					}
				}
			}

			// Get container status
			var containerStatus *corev1.ContainerStatus
			if pod.Status.ContainerStatuses != nil {
				for _, status := range pod.Status.ContainerStatuses {
					if status.Name == container.Name {
						containerStatus = &status
						break
					}
				}
			}

			// Build container info string
			containerInfo := fmt.Sprintf("name=%s;image=%s", container.Name, container.Image)

			// Add resource requests and limits
			if container.Resources.Requests != nil {
				for resourceName, quantity := range container.Resources.Requests {
					containerInfo += fmt.Sprintf(";request_%s=%s", resourceName, quantity.String())
				}
			}
			if container.Resources.Limits != nil {
				for resourceName, quantity := range container.Resources.Limits {
					containerInfo += fmt.Sprintf(";limit_%s=%s", resourceName, quantity.String())
				}
			}

			if containerStatus != nil {
				containerInfo += fmt.Sprintf(";ready=%v;restartCount=%d;started=%v",
					containerStatus.Ready,
					containerStatus.RestartCount,
					containerStatus.Started)

				// Container state
				if containerStatus.State.Running != nil {
					containerInfo += fmt.Sprintf(";state=Running;startedAt=%s",
						containerStatus.State.Running.StartedAt.String())
				} else if containerStatus.State.Waiting != nil {
					containerInfo += fmt.Sprintf(";state=Waiting;reason=%s;message=%s",
						containerStatus.State.Waiting.Reason,
						containerStatus.State.Waiting.Message)
				} else if containerStatus.State.Terminated != nil {
					containerInfo += fmt.Sprintf(";state=Terminated;exitCode=%d;reason=%s;message=%s;finishedAt=%s",
						containerStatus.State.Terminated.ExitCode,
						containerStatus.State.Terminated.Reason,
						containerStatus.State.Terminated.Message,
						containerStatus.State.Terminated.FinishedAt.String())
				}
			}

			containerStatuses = append(containerStatuses, containerInfo)

			// Capture container security context
			if container.SecurityContext != nil {
				containerSecCtx := map[string]interface{}{
					"containerName":          container.Name,
					"runAsUser":              container.SecurityContext.RunAsUser,
					"runAsGroup":             container.SecurityContext.RunAsGroup,
					"privileged":             container.SecurityContext.Privileged,
					"readOnlyRootFilesystem": container.SecurityContext.ReadOnlyRootFilesystem,
				}

				// Add additional security context fields if they exist
				if container.SecurityContext.AllowPrivilegeEscalation != nil {
					containerSecCtx["allowPrivilegeEscalation"] = *container.SecurityContext.AllowPrivilegeEscalation
				}
				if container.SecurityContext.RunAsNonRoot != nil {
					containerSecCtx["runAsNonRoot"] = *container.SecurityContext.RunAsNonRoot
				}
				if container.SecurityContext.Capabilities != nil {
					capabilities := map[string]interface{}{
						"add":  container.SecurityContext.Capabilities.Add,
						"drop": container.SecurityContext.Capabilities.Drop,
					}
					containerSecCtx["capabilities"] = capabilities
				}

				containerSecurityContexts = append(containerSecurityContexts, containerSecCtx)
			}
		}
	}

	// Convert resource quantities to strings
	requests := make(map[string]string)
	limits := make(map[string]string)
	for resourceName, quantity := range totalRequests {
		requests[string(resourceName)] = quantity.String()
	}
	for resourceName, quantity := range totalLimits {
		limits[string(resourceName)] = quantity.String()
	}

	// Handle potentially nil fields
	var startTimeStr string
	if pod.Status.StartTime != nil {
		startTimeStr = pod.Status.StartTime.String()
	}

	// Capture pod-level security context
	var podSecurityContext map[string]interface{}
	if pod.Spec.SecurityContext != nil {
		podSecurityContext = map[string]interface{}{
			"runAsUser":  pod.Spec.SecurityContext.RunAsUser,
			"runAsGroup": pod.Spec.SecurityContext.RunAsGroup,
			"fsGroup":    pod.Spec.SecurityContext.FSGroup,
		}

		// Add additional pod security context fields if they exist
		if pod.Spec.SecurityContext.RunAsNonRoot != nil {
			podSecurityContext["runAsNonRoot"] = *pod.Spec.SecurityContext.RunAsNonRoot
		}
		if pod.Spec.SecurityContext.SupplementalGroups != nil {
			podSecurityContext["supplementalGroups"] = pod.Spec.SecurityContext.SupplementalGroups
		}
		if pod.Spec.SecurityContext.Sysctls != nil {
			sysctls := make([]map[string]interface{}, 0)
			for _, sysctl := range pod.Spec.SecurityContext.Sysctls {
				sysctls = append(sysctls, map[string]interface{}{
					"name":  sysctl.Name,
					"value": sysctl.Value,
				})
			}
			podSecurityContext["sysctls"] = sysctls
		}
	}

	properties := map[string]interface{}{
		"name":                      pod.Name,
		"uid":                       string(pod.UID),
		"namespace":                 pod.Namespace,
		"nodeName":                  pod.Spec.NodeName,
		"creationTimestamp":         pod.CreationTimestamp.String(),
		"labels":                    pod.Labels,
		"annotations":               pod.Annotations,
		"status":                    string(pod.Status.Phase),
		"hostIP":                    pod.Status.HostIP,
		"podIP":                     pod.Status.PodIP,
		"startTime":                 startTimeStr,
		"priority":                  pod.Spec.Priority,
		"priorityClassName":         pod.Spec.PriorityClassName,
		"serviceAccount":            pod.Spec.ServiceAccountName,
		"restartPolicy":             string(pod.Spec.RestartPolicy),
		"conditions":                conditions,
		"resourceRequests":          requests,
		"resourceLimits":            limits,
		"containers":                containerStatuses,
		"containerSecurityContexts": containerSecurityContexts,
		"podSecurityContext":        podSecurityContext,
		"nodeSelector":              pod.Spec.NodeSelector,
		"tolerations":               formatTolerations(pod.Spec.Tolerations),
		"clusterName":               h.clusterName,
		"instanceHash":              h.instanceHash,
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Pod"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert pod %s: %w", pod.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if pod.OwnerReferences != nil {
		for _, ownerRef := range pod.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Pod", "uid", string(pod.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Pod %s and %s %s: %v\n", pod.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships
	if pod.Spec.NodeName != "" {
		if err := neo4jClient.CreateRelationship(ctx, "Pod", "uid", string(pod.UID), "SCHEDULED_ON", "Node", "name", pod.Spec.NodeName); err != nil {
			return fmt.Errorf("failed to create relationship between pod %s and node %s: %w", pod.Name, pod.Spec.NodeName, err)
		}
	}

	// Create relationships with PVCs
	if pod.Spec.Volumes != nil {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				if err := neo4jClient.CreateRelationship(ctx, "Pod", "uid", string(pod.UID), "USES", "PersistentVolumeClaim", "name", volume.PersistentVolumeClaim.ClaimName); err != nil {
					return fmt.Errorf("failed to create relationship between pod %s and PVC %s: %w", pod.Name, volume.PersistentVolumeClaim.ClaimName, err)
				}
			}
		}
	}

	// Create relationships with ConfigMaps
	if pod.Spec.Volumes != nil {
		for _, volume := range pod.Spec.Volumes {
			if volume.ConfigMap != nil {
				if err := neo4jClient.CreateRelationship(ctx, "Pod", "uid", string(pod.UID), "USES", "ConfigMap", "name", volume.ConfigMap.Name); err != nil {
					return fmt.Errorf("failed to create relationship between pod %s and ConfigMap %s: %w", pod.Name, volume.ConfigMap.Name, err)
				}
			}
		}
	}

	// Create relationships with Secrets
	if pod.Spec.Volumes != nil {
		for _, volume := range pod.Spec.Volumes {
			if volume.Secret != nil {
				if err := neo4jClient.CreateRelationship(ctx, "Pod", "uid", string(pod.UID), "USES", "Secret", "name", volume.Secret.SecretName); err != nil {
					return fmt.Errorf("failed to create relationship between pod %s and Secret %s: %w", pod.Name, volume.Secret.SecretName, err)
				}
			}
		}
	}

	return nil
}

func (h *PodHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	pod, err := ConvertToTyped[*corev1.Pod](obj)
	if err != nil {
		return fmt.Errorf("failed to convert pod: %w", err)
	}
	return HandleResourceDelete(ctx, "Pod", string(pod.UID), neo4jClient)
}

func formatTolerations(tolerations []corev1.Toleration) []string {
	result := make([]string, 0, len(tolerations))
	for _, t := range tolerations {
		s := fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect)
		if t.Operator != "" {
			s = fmt.Sprintf("%s(%s)", s, t.Operator)
		}
		result = append(result, s)
	}
	return result
}
