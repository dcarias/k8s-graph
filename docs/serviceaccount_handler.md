# ServiceAccount Handler

## Overview

The ServiceAccount handler is responsible for tracking Kubernetes ServiceAccount resources in the Neo4j graph database. ServiceAccounts provide an identity for processes that run in a Pod and are used for authentication and authorization.

## Resource Type

- **API Group**: `core/v1`
- **Resource**: `serviceaccounts`
- **Kind**: `ServiceAccount`

## Properties Stored

The handler stores the following properties for each ServiceAccount:

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | The name of the service account |
| `uid` | string | Unique identifier for the service account |
| `namespace` | string | The namespace containing the service account |
| `creationTimestamp` | string | When the service account was created |
| `labels` | map[string]string | Labels applied to the service account |
| `annotations` | map[string]string | Annotations applied to the service account |
| `secrets` | []ObjectReference | References to secrets associated with the service account |
| `imagePullSecrets` | []LocalObjectReference | References to image pull secrets |
| `automountToken` | *bool | Whether to automatically mount the service account token |
| `clusterName` | string | Name of the Kubernetes cluster |
| `instanceHash` | string | Hash identifying the kubegraph instance |

## Neo4j Node Structure

```cypher
(:ServiceAccount {
  name: "default",
  uid: "12345678-1234-1234-1234-123456789abc",
  namespace: "default",
  creationTimestamp: "2024-01-01T00:00:00Z",
  labels: {app: "my-app"},
  annotations: {description: "Default service account"},
  secrets: [{name: "default-token-abc123", uid: "secret-uid"}],
  imagePullSecrets: [{name: "my-registry-secret"}],
  automountToken: true,
  clusterName: "my-cluster",
  instanceHash: "abc123"
})
```

## Relationships

### Owner References

The handler creates `OWNED_BY` relationships to parent resources based on owner references:

```cypher
(:ServiceAccount)-[:OWNED_BY]->(:ParentResource)
```

### Example Queries

#### List all service accounts in a namespace

```cypher
MATCH (sa:ServiceAccount {namespace: "default"})
RETURN sa.name, sa.creationTimestamp, sa.automountToken
ORDER BY sa.name
```

#### Find service accounts with specific labels

```cypher
MATCH (sa:ServiceAccount)
WHERE sa.labels.app = "my-app"
RETURN sa.name, sa.namespace, sa.labels
```

#### Get service account hierarchy with owner references

```cypher
MATCH (sa:ServiceAccount)-[:OWNED_BY]->(owner)
WHERE sa.name = "my-serviceaccount"
RETURN sa.name, owner.name, labels(owner)[0] as ownerType
```

#### Count service accounts per namespace

```cypher
MATCH (sa:ServiceAccount)
RETURN sa.namespace, count(sa) as serviceAccountCount
ORDER BY serviceAccountCount DESC
```

#### Find service accounts with image pull secrets

```cypher
MATCH (sa:ServiceAccount)
WHERE size(sa.imagePullSecrets) > 0
RETURN sa.name, sa.namespace, sa.imagePullSecrets
```

#### Find service accounts with automount token disabled

```cypher
MATCH (sa:ServiceAccount)
WHERE sa.automountToken = false
RETURN sa.name, sa.namespace
```

#### Get service accounts with their associated secrets

```cypher
MATCH (sa:ServiceAccount)
WHERE size(sa.secrets) > 0
RETURN sa.name, sa.namespace, 
       [secret IN sa.secrets | secret.name] as secretNames
```

## Implementation Details

### Handler Registration

The ServiceAccount handler is automatically registered in the Kubernetes client:

```go
handlers.NewServiceAccountHandler(c.config)
```

### Event Handling

- **Create/Update**: Creates or updates the ServiceAccount node and establishes owner reference relationships
- **Delete**: Removes the ServiceAccount node and all its relationships from the graph

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

1. **Authentication Analysis**: Track service account usage across the cluster
2. **Security Auditing**: Monitor service account permissions and token usage
3. **Resource Access**: Analyze which resources use specific service accounts
4. **Secret Management**: Track service account secret associations
5. **Image Registry Access**: Monitor image pull secret usage

## Security Considerations

- Service accounts are critical for authentication and authorization
- The handler tracks sensitive information like secret references
- Consider access controls when querying service account data
- Monitor for unusual service account creation or modification patterns

## Related Resources

- [Kubernetes ServiceAccounts Documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
- [Neo4j Relationships Documentation](docs/neo4j_relationships.md)
- [Neo4j Ownership Queries](docs/neo4j_ownership_queries.md) 
