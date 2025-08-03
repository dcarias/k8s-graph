# Namespace Handler

## Overview

The Namespace handler is responsible for tracking Kubernetes Namespace resources in the Neo4j graph database. Namespaces provide a mechanism for isolating groups of resources within a single cluster.

## Resource Type

- **API Group**: `core/v1`
- **Resource**: `namespaces`
- **Kind**: `Namespace`

## Properties Stored

The handler stores the following properties for each Namespace:

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | The name of the namespace |
| `uid` | string | Unique identifier for the namespace |
| `creationTimestamp` | string | When the namespace was created |
| `labels` | map[string]string | Labels applied to the namespace |
| `annotations` | map[string]string | Annotations applied to the namespace |
| `status` | string | The current phase of the namespace (Active, Terminating, etc.) |
| `clusterName` | string | Name of the Kubernetes cluster |
| `instanceHash` | string | Hash identifying the kubegraph instance |

## Neo4j Node Structure

```cypher
(:Namespace {
  name: "default",
  uid: "12345678-1234-1234-1234-123456789abc",
  creationTimestamp: "2024-01-01T00:00:00Z",
  labels: {environment: "production"},
  annotations: {description: "Default namespace"},
  status: "Active",
  clusterName: "my-cluster",
  instanceHash: "abc123"
})
```

## Relationships

### Owner References

The handler creates `OWNED_BY` relationships to parent resources based on owner references:

```cypher
(:Namespace)-[:OWNED_BY]->(:ParentResource)
```

### Example Queries

#### List all namespaces in a cluster

```cypher
MATCH (n:Namespace {clusterName: "my-cluster"})
RETURN n.name, n.status, n.creationTimestamp
ORDER BY n.name
```

#### Find namespaces with specific labels

```cypher
MATCH (n:Namespace)
WHERE n.labels.environment = "production"
RETURN n.name, n.labels
```

#### Get namespace hierarchy with owner references

```cypher
MATCH (n:Namespace)-[:OWNED_BY]->(owner)
WHERE n.name = "my-namespace"
RETURN n.name, owner.name, labels(owner)[0] as ownerType
```

#### Count resources per namespace

```cypher
MATCH (n:Namespace)
OPTIONAL MATCH (n)<-[:IN_NAMESPACE]-(resource)
WHERE labels(resource)[0] IN ["Pod", "Service", "Deployment"]
RETURN n.name, 
       count(DISTINCT CASE WHEN labels(resource)[0] = "Pod" THEN resource END) as pods,
       count(DISTINCT CASE WHEN labels(resource)[0] = "Service" THEN resource END) as services,
       count(DISTINCT CASE WHEN labels(resource)[0] = "Deployment" THEN resource END) as deployments
ORDER BY n.name
```

#### Find namespaces with specific status

```cypher
MATCH (n:Namespace)
WHERE n.status = "Terminating"
RETURN n.name, n.creationTimestamp
```

## Implementation Details

### Handler Registration

The Namespace handler is automatically registered in the Kubernetes client:

```go
handlers.NewNamespaceHandler(c.config)
```

### Event Handling

- **Create/Update**: Creates or updates the Namespace node and establishes owner reference relationships
- **Delete**: Removes the Namespace node and all its relationships from the graph

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

1. **Namespace Management**: Track namespace lifecycle and status
2. **Resource Organization**: Understand resource distribution across namespaces
3. **Access Control**: Analyze namespace-based access patterns
4. **Resource Quotas**: Monitor resource usage per namespace
5. **Multi-tenancy**: Support multi-tenant cluster analysis

## Related Resources

- [Kubernetes Namespaces Documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)
- [Neo4j Relationships Documentation](docs/neo4j_relationships.md)
- [Neo4j Ownership Queries](docs/neo4j_ownership_queries.md) 
