# VerticalPodAutoscaler Handler

## Overview

The VerticalPodAutoscaler (VPA) handler is responsible for tracking Kubernetes VerticalPodAutoscaler resources in the Neo4j graph database. VPAs automatically adjust the CPU and memory resource requests and limits for pods based on usage patterns, optimizing resource utilization.

## Resource Type

- **API Group**: `autoscaling.k8s.io/v1`
- **Resource**: `verticalpodautoscalers`
- **Kind**: `VerticalPodAutoscaler`

## Properties Stored

The handler stores the following properties for each VerticalPodAutoscaler:

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | The name of the VPA |
| `uid` | string | Unique identifier for the VPA |
| `namespace` | string | The namespace containing the VPA |
| `creationTimestamp` | string | When the VPA was created |
| `labels` | map[string]string | Labels applied to the VPA |
| `annotations` | map[string]string | Annotations applied to the VPA |
| `spec` | map[string]interface{} | The complete VPA specification |
| `status` | map[string]interface{} | The current VPA status |
| `clusterName` | string | Name of the Kubernetes cluster |
| `instanceHash` | string | Hash identifying the kubegraph instance |

## Neo4j Node Structure

```cypher
(:VerticalPodAutoscaler {
  name: "my-app-vpa",
  uid: "12345678-1234-1234-1234-123456789abc",
  namespace: "default",
  creationTimestamp: "2024-01-01T00:00:00Z",
  labels: {app: "my-app"},
  annotations: {description: "VPA for my application"},
  spec: {
    targetRef: {apiVersion: "apps/v1", kind: "Deployment", name: "my-app"},
    updatePolicy: {updateMode: "Auto"},
    resourcePolicy: {containerPolicies: [{containerName: "*", minAllowed: {cpu: "100m", memory: "50Mi"}, maxAllowed: {cpu: "1", memory: "500Mi"}}]}
  },
  status: {
    recommendations: [{containerRecommendations: [{containerName: "app", target: {cpu: "250m", memory: "128Mi"}, lowerBound: {cpu: "200m", memory: "100Mi"}, upperBound: {cpu: "300m", memory: "150Mi"}}]}]
  },
  clusterName: "my-cluster",
  instanceHash: "abc123"
})
```

## Relationships

### Owner References

The handler creates `OWNED_BY` relationships to parent resources based on owner references:

```cypher
(:VerticalPodAutoscaler)-[:OWNED_BY]->(:ParentResource)
```

### Scale Target

The handler creates `SCALES` relationships to the target resource being scaled:

```cypher
(:VerticalPodAutoscaler)-[:SCALES]->(:TargetResource)
```

### Example Queries

#### List all VPAs in a namespace

```cypher
MATCH (vpa:VerticalPodAutoscaler {namespace: "default"})
RETURN vpa.name, vpa.spec.targetRef, vpa.spec.updatePolicy
ORDER BY vpa.name
```

#### Find VPAs with specific scaling targets

```cypher
MATCH (vpa:VerticalPodAutoscaler)-[:SCALES]->(target)
WHERE target.name = "my-deployment"
RETURN vpa.name, vpa.namespace, labels(target)[0] as targetType
```

#### Get VPA scaling statistics

```cypher
MATCH (vpa:VerticalPodAutoscaler)
RETURN vpa.namespace,
       count(vpa) as vpaCount,
       [vpa IN collect(vpa) | vpa.spec.updatePolicy.updateMode] as updateModes
ORDER BY vpaCount DESC
```

#### Find VPAs with Auto update mode

```cypher
MATCH (vpa:VerticalPodAutoscaler)
WHERE vpa.spec.updatePolicy.updateMode = "Auto"
RETURN vpa.name, vpa.namespace, vpa.spec.targetRef
```

#### Find VPAs with resource policies

```cypher
MATCH (vpa:VerticalPodAutoscaler)
WHERE vpa.spec.resourcePolicy IS NOT NULL
RETURN vpa.name, vpa.namespace, vpa.spec.resourcePolicy
```

#### Get VPA hierarchy with owner references

```cypher
MATCH (vpa:VerticalPodAutoscaler)-[:OWNED_BY]->(owner)
WHERE vpa.name = "my-vpa"
RETURN vpa.name, owner.name, labels(owner)[0] as ownerType
```

#### Find VPAs with recommendations

```cypher
MATCH (vpa:VerticalPodAutoscaler)
WHERE vpa.status.recommendations IS NOT NULL
RETURN vpa.name, vpa.namespace, vpa.status.recommendations
```

#### Analyze VPA target types

```cypher
MATCH (vpa:VerticalPodAutoscaler)
WHERE vpa.spec.targetRef IS NOT NULL
RETURN vpa.spec.targetRef.kind, count(*) as targetCount
ORDER BY targetCount DESC
```

#### Find VPAs with specific resource constraints

```cypher
MATCH (vpa:VerticalPodAutoscaler)
WHERE ANY(policy IN vpa.spec.resourcePolicy.containerPolicies WHERE policy.maxAllowed.cpu IS NOT NULL)
RETURN vpa.name, vpa.namespace, 
       [policy IN vpa.spec.resourcePolicy.containerPolicies WHERE policy.maxAllowed.cpu IS NOT NULL | policy.maxAllowed.cpu] as maxCPU
```

## Implementation Details

### Handler Registration

The VerticalPodAutoscaler handler is automatically registered in the Kubernetes client:

```go
handlers.NewVerticalPodAutoscalerHandler(c.config)
```

### Event Handling

- **Create/Update**: Creates or updates the VPA node, establishes owner reference relationships, and creates SCALES relationships to target resources
- **Delete**: Removes the VPA node and all its relationships from the graph

### Error Handling

The handler includes comprehensive error handling for:
- Object conversion failures
- Neo4j operation failures
- Relationship creation failures

### Performance Considerations

- Uses upsert operations to handle both create and update events efficiently
- Implements proper cleanup on deletion to maintain graph consistency
- Leverages Neo4j's indexing on the `uid` property for fast lookups
- Works with unstructured objects since VPA is not part of the standard Kubernetes API

## Use Cases

1. **Resource Optimization**: Monitor vertical scaling recommendations and patterns
2. **Cost Management**: Track resource usage optimization through VPA recommendations
3. **Performance Analysis**: Analyze resource allocation patterns and recommendations
4. **Capacity Planning**: Understand resource requirements based on VPA recommendations
5. **Resource Efficiency**: Monitor how well applications utilize allocated resources

## VPA Modes

The handler tracks various VPA update modes:
- **Auto**: Automatically applies recommendations
- **Initial**: Only applies recommendations at pod creation
- **Off**: Only provides recommendations without applying them

## Resource Policies

The handler tracks various resource policies:
- **Container Policies**: Per-container resource constraints
- **Min Allowed**: Minimum resource requests and limits
- **Max Allowed**: Maximum resource requests and limits
- **Controlled Resources**: Which resources are controlled by VPA

## Important Notes

- **VPA Installation Required**: VPA must be installed in the cluster for this handler to work
- **Unstructured Objects**: Since VPA is not part of the standard Kubernetes API, the handler works with unstructured objects
- **API Group**: Uses `autoscaling.k8s.io/v1` API group
- **Compatibility**: Works with VPA installations that follow the standard VPA API structure

## Related Resources

- [Kubernetes VPA Documentation](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
- [Neo4j Relationships Documentation](docs/neo4j_relationships.md)
- [Neo4j Ownership Queries](docs/neo4j_ownership_queries.md) 
