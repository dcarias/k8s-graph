package handlers

import (
	"context"
	"fmt"
	"time"

	"k8s-graph/config"
	"k8s-graph/pkg/neo4j"

	driverneo4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type EventHandler struct {
	BaseHandler
}

func NewEventHandler(cfg *config.Config) *EventHandler {
	gvr := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "events",
	}
	RegisterOwnerKind("Event", "Event")
	return &EventHandler{
		BaseHandler: NewBaseHandler(gvr, "Event", cfg),
	}
}

func (h *EventHandler) HandleCreate(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	event, err := ConvertToTyped[*corev1.Event](obj)
	if err != nil {
		return fmt.Errorf("failed to convert event: %w", err)
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)
	properties := map[string]interface{}{
		"name":           event.Name,
		"uid":            string(event.UID),
		"namespace":      event.Namespace,
		"reason":         event.Reason,
		"message":        event.Message,
		"type":           event.Type,
		"count":          event.Count,
		"firstTimestamp": event.FirstTimestamp.String(),
		"lastTimestamp":  event.LastTimestamp.String(),
		"eventTime":      event.EventTime.String(),
		"source":         event.Source,
		"involvedObject": event.InvolvedObject,
		"labels":         event.Labels,
		"annotations":    event.Annotations,
		"createdAt":      createdAt,
		"clusterName":    h.GetClusterName(),
		// Events should not have instanceHash as they should persist across restarts
	}

	if err := neo4jClient.UpsertNode(ctx, []string{"Event"}, properties, "uid"); err != nil {
		return fmt.Errorf("failed to upsert event %s: %w", event.Name, err)
	}

	// Relationship to involved object
	if event.InvolvedObject.UID != "" && event.InvolvedObject.Kind != "" {
		if label, ok := ownerKindToLabel[event.InvolvedObject.Kind]; ok {
			err := neo4jClient.CreateRelationship(
				ctx,
				"Event", "uid", string(event.UID),
				"INVOLVES",
				label, "uid", string(event.InvolvedObject.UID),
			)
			if err != nil {
				fmt.Printf("Warning: failed to create INVOLVES relationship for Event %s: %v\n", event.Name, err)
			}
		}
	}

	return nil
}

func (h *EventHandler) HandleDelete(ctx context.Context, obj interface{}, neo4jClient *neo4j.Client) error {
	// Do not delete events from Neo4j when Kubernetes deletes them
	// Events should persist in Neo4j and only be cleaned up via TTL mechanism
	// using PruneExpiredEvents function
	return nil
}

// PruneExpiredEvents deletes Event nodes older than the given TTL (in days)
func PruneExpiredEvents(ctx context.Context, neo4jClient *neo4j.Client, ttlDays int) error {
	if ttlDays <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().Add(-time.Duration(ttlDays) * 24 * time.Hour).Format(time.RFC3339)
	query := `MATCH (e:Event) WHERE e.createdAt < $cutoff DETACH DELETE e`
	params := map[string]interface{}{"cutoff": cutoff}
	session := neo4jClient.Driver().NewSession(ctx, driverneo4j.SessionConfig{AccessMode: driverneo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.Run(ctx, query, params)
	return err
}