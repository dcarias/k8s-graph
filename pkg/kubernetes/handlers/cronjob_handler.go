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

type CronJobHandler struct {
	BaseHandler
	clientset    *kubernetes.Clientset
	instanceHash string
}

func NewCronJobHandler(clientset *kubernetes.Clientset, cfg *config.Config) *CronJobHandler {
	gvr := schema.GroupVersionResource{
		Group:    "batch",
		Version:  "v1",
		Resource: "cronjobs",
	}
	// Register this handler's kind for owner references
	RegisterOwnerKind("CronJob", "CronJob")
	return &CronJobHandler{
		BaseHandler:  NewBaseHandler(gvr, "CronJob", cfg),
		clientset:    clientset,
		instanceHash: cfg.InstanceHash,
	}
}

func (h *CronJobHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	cronjob, err := ConvertToTyped[*batchv1.CronJob](obj)
	if err != nil {
		return fmt.Errorf("failed to convert cronjob: %w", err)
	}

	properties := map[string]interface{}{
		"name":                       cronjob.Name,
		"uid":                        string(cronjob.UID),
		"namespace":                  cronjob.Namespace,
		"creationTimestamp":          cronjob.CreationTimestamp.String(),
		"labels":                     cronjob.Labels,
		"annotations":                cronjob.Annotations,
		"schedule":                   cronjob.Spec.Schedule,
		"timeZone":                   cronjob.Spec.TimeZone,
		"startingDeadlineSeconds":    cronjob.Spec.StartingDeadlineSeconds,
		"concurrencyPolicy":          string(cronjob.Spec.ConcurrencyPolicy),
		"successfulJobsHistoryLimit": cronjob.Spec.SuccessfulJobsHistoryLimit,
		"failedJobsHistoryLimit":     cronjob.Spec.FailedJobsHistoryLimit,
		"suspend":                    cronjob.Spec.Suspend,
		"clusterName":                h.GetClusterName(),
		"instanceHash":               h.instanceHash,
	}

	// Add status information
	if cronjob.Status.LastScheduleTime != nil {
		properties["lastScheduleTime"] = cronjob.Status.LastScheduleTime.String()
	}
	if cronjob.Status.LastSuccessfulTime != nil {
		properties["lastSuccessfulTime"] = cronjob.Status.LastSuccessfulTime.String()
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"CronJob"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert cronjob %s: %w", cronjob.Name, err)
	}

	// Create relationships based on owner references for all supported types
	if cronjob.OwnerReferences != nil {
		for _, ownerRef := range cronjob.OwnerReferences {
			if label, ok := ownerKindToLabel[ownerRef.Kind]; ok {
				err := neo4jClient.CreateRelationship(
					ctx,
					"CronJob", "uid", string(cronjob.UID),
					"OWNED_BY",
					label, "uid", string(ownerRef.UID),
				)
				if err != nil {
					fmt.Printf("Warning: failed to create relationship between CronJob %s and %s %s: %v\n", cronjob.Name, label, ownerRef.Name, err)
				}
			}
		}
	}

	// Create relationships with jobs created by this cronjob
	// Use a more robust approach to find jobs owned by this cronjob
	jobs, err := h.clientset.BatchV1().Jobs(cronjob.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list jobs for cronjob %s: %w", cronjob.Name, err)
	}

	for _, job := range jobs.Items {
		// Check if this job is owned by this cronjob
		for _, ownerRef := range job.OwnerReferences {
			if ownerRef.Kind == "CronJob" && ownerRef.UID == cronjob.UID {
				if err := neo4jClient.CreateRelationship(ctx, "CronJob", "uid", string(cronjob.UID), "CREATES", "Job", "uid", string(job.UID)); err != nil {
					return fmt.Errorf("failed to create relationship between cronjob %s and job %s: %w", cronjob.Name, job.Name, err)
				}
				break
			}
		}
	}

	return nil
}

func (h *CronJobHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	cronjob, err := ConvertToTyped[*batchv1.CronJob](obj)
	if err != nil {
		return fmt.Errorf("failed to convert cronjob: %w", err)
	}
	return HandleResourceDelete(ctx, "CronJob", string(cronjob.UID), neo4jClient)
}
