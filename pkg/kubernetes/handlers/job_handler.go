package handlers

import (
	"context"
	"fmt"

	"kubegraph/config"
	"kubegraph/pkg/neo4j"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type JobHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewJobHandler(clientset *kubernetes.Clientset, cfg *config.Config) *JobHandler {
	gvr := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "jobs",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("Job", "Job")
	return &JobHandler{
		BaseHandler:  NewBaseHandler(gvr, "Job", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *JobHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	job, err := ConvertToTyped[*batchv1.Job](obj)
	if err != nil {
		return fmt.Errorf("failed to convert job: %w", err)
	}

	properties := map[string]interface{}{
		"name":                    job.Name,
		"uid":                     string(job.UID),
		"namespace":               job.Namespace,
		"creationTimestamp":       job.CreationTimestamp.String(),
		"labels":                  job.Labels,
		"annotations":             job.Annotations,
		"parallelism":             job.Spec.Parallelism,
		"completions":             job.Spec.Completions,
		"backoffLimit":            job.Spec.BackoffLimit,
		"activeDeadlineSeconds":   job.Spec.ActiveDeadlineSeconds,
		"ttlSecondsAfterFinished": job.Spec.TTLSecondsAfterFinished,
		"clusterName":             h.GetClusterName(),
		"instanceHash":            h.instanceHash,
	}

	// Add status information
	properties["active"] = job.Status.Active
	properties["succeeded"] = job.Status.Succeeded
	properties["failed"] = job.Status.Failed
	if job.Status.StartTime != nil {
		properties["startTime"] = job.Status.StartTime.String()
	}
	if job.Status.CompletionTime != nil {
		properties["completionTime"] = job.Status.CompletionTime.String()
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Job"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert job %s: %w", job.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if job.OwnerReferences != nil {
		for _, ownerRef := range job.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"Job", "uid", string(job.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between Job %s and %s %s: %v\n", job.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with pods
	if job.Spec.Selector != nil {
		pods, err := h.clientset.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(job.Spec.Selector),
		})
		if err != nil {
			return fmt.Errorf("failed to list pods for job %s: %w", job.Name, err)
		}

		for _, pod := range pods.Items {
			if err := neo4jClient.CreateRelationship(ctx, "Job", "uid", string(job.UID), "MANAGES", "Pod", "uid", string(pod.UID)); err != nil {
				return fmt.Errorf("failed to create relationship between job %s and pod %s: %w", job.Name, pod.Name, err)
			}
		}
	}

	return nil
}

func (h *JobHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	job, err := ConvertToTyped[*batchv1.Job](obj)
	if err != nil {
		return fmt.Errorf("failed to convert job: %w", err)
	}
	return HandleResourceDelete(ctx, "Job", string(job.UID), neo4jClient)
}
