# HorizontalPodAutoscaler Handler

## Overview

The HorizontalPodAutoscaler (HPA) handler is responsible for tracking Kubernetes HorizontalPodAutoscaler resources in the Neo4j graph database. HPAs automatically scale the number of pods in a replication controller, deployment, replica set, or stateful set based on observed CPU utilization or other custom metrics.

## Resource Type

- **API Group**: `autoscaling/v2`
- **Resource**: `horizontalpodautoscalers`
- **Kind**: `HorizontalPodAutoscaler`

## Properties Stored

The handler stores the following properties for each HorizontalPodAutoscaler:

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | The name of the HPA |
| `uid` | string | Unique identifier for the HPA |
| `namespace` | string | The namespace containing the HPA |
| `creationTimestamp` | string | When the HPA was created |
| `labels` | map[string]string | Labels applied to the HPA |
| `annotations` | map[string]string | Annotations applied to the HPA |
| `minReplicas` | *int32 | Minimum number of replicas |
| `maxReplicas` | int32 | Maximum number of replicas |
| `scaleTargetRef` | CrossVersionObjectReference | Reference to the target resource to scale |
| `metrics` | []MetricSpec | List of metrics used for scaling |
| `behavior` | *HorizontalPodAutoscalerBehavior | Scaling behavior configuration |
| `currentReplicas` | int32 | Current number of replicas |
| `desiredReplicas` | int32 | Desired number of replicas |
| `currentMetrics` | []MetricStatus | Current metric values |
| `conditions` | []HorizontalPodAutoscalerCondition | Current conditions |
| `lastScaleTime` | *metav1.Time | Last time the HPA scaled the target |
| `clusterName` | string | Name of the Kubernetes cluster |
| `instanceHash` | string | Hash identifying the kubegraph instance |

## Neo4j Node Structure

```cypher
(:HorizontalPodAutoscaler {
  name: "my-app-hpa",
  uid: "12345678-1234-1234-1234-123456789abc",
  namespace: "default",
  creationTimestamp: "2024-01-01T00:00:00Z",
  labels: {app: "my-app"},
  annotations: {description: "HPA for my application"},
  minReplicas: 2,
  maxReplicas: 10,
  scaleTargetRef: {apiVersion: "apps/v1", kind: "Deployment", name: "my-app"},
  metrics: [{type: "Resource", resource: {name: "cpu", target: {type: "Utilization", averageUtilization: 70}}}],
  currentReplicas: 3,
  desiredReplicas: 3,
  lastScaleTime: "2024-01-01T12:00:00Z",
  clusterName: "my-cluster",
  instanceHash: "abc123"
})
```

## Relationships

### Owner References

The handler creates `OWNED_BY` relationships to parent resources based on owner references:

```cypher
(:HorizontalPodAutoscaler)-[:OWNED_BY]->(:ParentResource)
```

### Scale Target

The handler creates `SCALES` relationships to the target resource being scaled:

```cypher
(:HorizontalPodAutoscaler)-[:SCALES]->(:TargetResource)
```

### Example Queries

#### List all HPAs in a namespace

```cypher
MATCH (hpa:HorizontalPodAutoscaler {namespace: "default"})
RETURN hpa.name, hpa.minReplicas, hpa.maxReplicas, hpa.currentReplicas, hpa.desiredReplicas
ORDER BY hpa.name
```

#### Find HPAs with specific scaling targets

```cypher
MATCH (hpa:HorizontalPodAutoscaler)-[:SCALES]->(target)
WHERE target.name = "my-deployment"
RETURN hpa.name, hpa.namespace, labels(target)[0] as targetType
```

#### Get HPA scaling statistics

```cypher
MATCH (hpa:HorizontalPodAutoscaler)
RETURN hpa.namespace,
       count(hpa) as hpaCount,
       avg(hpa.maxReplicas) as avgMaxReplicas,
       avg(hpa.currentReplicas) as avgCurrentReplicas
ORDER BY hpaCount DESC
```

#### Find HPAs that have scaled recently

```cypher
MATCH (hpa:HorizontalPodAutoscaler)
WHERE hpa.lastScaleTime IS NOT NULL
RETURN hpa.name, hpa.namespace, hpa.lastScaleTime, hpa.currentReplicas, hpa.desiredReplicas
ORDER BY hpa.lastScaleTime DESC
```

#### Find HPAs with CPU-based scaling

```cypher
MATCH (hpa:HorizontalPodAutoscaler)
WHERE ANY(metric IN hpa.metrics WHERE metric.type = "Resource" AND metric.resource.name = "cpu")
RETURN hpa.name, hpa.namespace, hpa.metrics
```

#### Get HPA hierarchy with owner references

```cypher
MATCH (hpa:HorizontalPodAutoscaler)-[:OWNED_BY]->(owner)
WHERE hpa.name = "my-hpa"
RETURN hpa.name, owner.name, labels(owner)[0] as ownerType
```

#### Find HPAs with high replica counts

```cypher
MATCH (hpa:HorizontalPodAutoscaler)
WHERE hpa.currentReplicas > 5
RETURN hpa.name, hpa.namespace, hpa.currentReplicas, hpa.maxReplicas
ORDER BY hpa.currentReplicas DESC
```

#### Analyze HPA scaling patterns

```cypher
MATCH (hpa:HorizontalPodAutoscaler)
WHERE hpa.currentReplicas != hpa.desiredReplicas
RETURN hpa.name, hpa.namespace, 
       hpa.currentReplicas, hpa.desiredReplicas,
       hpa.currentReplicas - hpa.desiredReplicas as scalingDifference
```

## Implementation Details

### Handler Registration

The HorizontalPodAutoscaler handler is automatically registered in the Kubernetes client:

```go
handlers.NewHorizontalPodAutoscalerHandler(c.config)
```

### Event Handling

- **Create/Update**: Creates or updates the HPA node, establishes owner reference relationships, and creates SCALES relationships to target resources
- **Delete**: Removes the HPA node and all its relationships from the graph

### Error Handling

The handler includes comprehensive error handling for:
- Object conversion failures
- Neo4j operation failures
- Relationship creation failures

### Performance Considerations

- Uses upsert operations to handle both create and update events efficiently
- Implements proper cleanup on deletion to maintain graph consistency
- Leverages Neo4j's indexing on the `uid` property for fast lookups

## Use Cases

1. **Scaling Analysis**: Monitor autoscaling behavior and patterns
2. **Performance Optimization**: Analyze scaling metrics and thresholds
3. **Resource Planning**: Understand scaling requirements across applications
4. **Cost Management**: Track resource usage and scaling costs
5. **Capacity Planning**: Plan for peak scaling requirements

## Scaling Metrics

The handler tracks various scaling metrics including:
- **Resource metrics**: CPU, memory utilization
- **Custom metrics**: Application-specific metrics
- **Object metrics**: Pod-based metrics
- **External metrics**: External system metrics

## Related Resources

- [Kubernetes HPA Documentation](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [Neo4j Relationships Documentation](docs/neo4j_relationships.md)
- [Neo4j Ownership Queries](docs/neo4j_ownership_queries.md) 
