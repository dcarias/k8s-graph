# LimitRange Handler

## Overview

The LimitRange handler is responsible for tracking Kubernetes LimitRange resources in the Neo4j graph database. LimitRanges provide default resource request and limit values for containers in a namespace, and enforce minimum and maximum resource constraints.

## Resource Type

- **API Group**: `core/v1`
- **Resource**: `limitranges`
- **Kind**: `LimitRange`

## Properties Stored

The handler stores the following properties for each LimitRange:

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | The name of the limit range |
| `uid` | string | Unique identifier for the limit range |
| `namespace` | string | The namespace containing the limit range |
| `creationTimestamp` | string | When the limit range was created |
| `labels` | map[string]string | Labels applied to the limit range |
| `annotations` | map[string]string | Annotations applied to the limit range |
| `spec` | LimitRangeSpec | The complete limit range specification |
| `clusterName` | string | Name of the Kubernetes cluster |
| `instanceHash` | string | Hash identifying the kubegraph instance |

## Neo4j Node Structure

```cypher
(:LimitRange {
  name: "default-limits",
  uid: "12345678-1234-1234-1234-123456789abc",
  namespace: "default",
  creationTimestamp: "2024-01-01T00:00:00Z",
  labels: {purpose: "resource-limits"},
  annotations: {description: "Default resource limits for namespace"},
  spec: {
    limits: [{
      type: "Container",
      default: {cpu: "500m", memory: "512Mi"},
      defaultRequest: {cpu: "250m", memory: "256Mi"},
      min: {cpu: "100m", memory: "128Mi"},
      max: {cpu: "2", memory: "2Gi"}
    }]
  },
  clusterName: "my-cluster",
  instanceHash: "abc123"
})
```

## Relationships

### Owner References

The handler creates `OWNED_BY` relationships to parent resources based on owner references:

```cypher
(:LimitRange)-[:OWNED_BY]->(:ParentResource)
```

### Example Queries

#### List all limit ranges in a namespace

```cypher
MATCH (lr:LimitRange {namespace: "default"})
RETURN lr.name, lr.creationTimestamp, lr.spec
ORDER BY lr.name
```

#### Find limit ranges with specific labels

```cypher
MATCH (lr:LimitRange)
WHERE lr.labels.purpose = "resource-limits"
RETURN lr.name, lr.namespace, lr.labels
```

#### Get limit range hierarchy with owner references

```cypher
MATCH (lr:LimitRange)-[:OWNED_BY]->(owner)
WHERE lr.name = "my-limitrange"
RETURN lr.name, owner.name, labels(owner)[0] as ownerType
```

#### Count limit ranges per namespace

```cypher
MATCH (lr:LimitRange)
RETURN lr.namespace, count(lr) as limitRangeCount
ORDER BY limitRangeCount DESC
```

#### Find limit ranges with CPU constraints

```cypher
MATCH (lr:LimitRange)
WHERE ANY(limit IN lr.spec.limits WHERE limit.max.cpu IS NOT NULL)
RETURN lr.name, lr.namespace, 
       [limit IN lr.spec.limits WHERE limit.max.cpu IS NOT NULL | limit.max.cpu] as cpuLimits
```

#### Find limit ranges with memory constraints

```cypher
MATCH (lr:LimitRange)
WHERE ANY(limit IN lr.spec.limits WHERE limit.max.memory IS NOT NULL)
RETURN lr.name, lr.namespace, 
       [limit IN lr.spec.limits WHERE limit.max.memory IS NOT NULL | limit.max.memory] as memoryLimits
```

#### Find limit ranges with default values

```cypher
MATCH (lr:LimitRange)
WHERE ANY(limit IN lr.spec.limits WHERE limit.default IS NOT NULL)
RETURN lr.name, lr.namespace, 
       [limit IN lr.spec.limits WHERE limit.default IS NOT NULL | limit.default] as defaults
```

#### Find limit ranges with minimum constraints

```cypher
MATCH (lr:LimitRange)
WHERE ANY(limit IN lr.spec.limits WHERE limit.min IS NOT NULL)
RETURN lr.name, lr.namespace, 
       [limit IN lr.spec.limits WHERE limit.min IS NOT NULL | limit.min] as minimums
```

#### Analyze limit range types

```cypher
MATCH (lr:LimitRange)
UNWIND lr.spec.limits as limit
RETURN lr.namespace, limit.type, count(*) as typeCount
ORDER BY lr.namespace, typeCount DESC
```

## Implementation Details

### Handler Registration

The LimitRange handler is automatically registered in the Kubernetes client:

```go
handlers.NewLimitRangeHandler(c.config)
```

### Event Handling

- **Create/Update**: Creates or updates the LimitRange node and establishes owner reference relationships
- **Delete**: Removes the LimitRange node and all its relationships from the graph

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

1. **Resource Management**: Monitor resource limits and requests across namespaces
2. **Cost Control**: Enforce resource constraints to control costs
3. **Capacity Planning**: Understand resource allocation patterns
4. **Compliance**: Ensure applications meet resource requirements
5. **Multi-tenancy**: Manage resource allocation in multi-tenant environments

## Limit Types

The handler tracks various limit types:
- **Container**: Limits for individual containers
- **Pod**: Limits for entire pods
- **PersistentVolumeClaim**: Limits for PVC storage requests

## Resource Constraints

The handler tracks various resource constraints:
- **CPU**: CPU requests and limits
- **Memory**: Memory requests and limits
- **Storage**: Storage requests and limits
- **Ephemeral Storage**: Temporary storage limits

## Related Resources

- [Kubernetes LimitRange Documentation](https://kubernetes.io/docs/concepts/policy/limit-range/)
- [Neo4j Relationships Documentation](docs/neo4j_relationships.md)
- [Neo4j Ownership Queries](docs/neo4j_ownership_queries.md) 
