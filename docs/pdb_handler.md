# PodDisruptionBudget Handler

## Overview

The PodDisruptionBudget (PDB) handler is responsible for tracking Kubernetes PodDisruptionBudget resources in the Neo4j graph database. PDBs limit the number of pods of a replicated application that are down simultaneously from voluntary disruptions, ensuring application availability during cluster maintenance.

## Resource Type

- **API Group**: `policy/v1`
- **Resource**: `poddisruptionbudgets`
- **Kind**: `PodDisruptionBudget`

## Properties Stored

The handler stores the following properties for each PodDisruptionBudget:

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | The name of the PDB |
| `uid` | string | Unique identifier for the PDB |
| `namespace` | string | The namespace containing the PDB |
| `creationTimestamp` | string | When the PDB was created |
| `labels` | map[string]string | Labels applied to the PDB |
| `annotations` | map[string]string | Annotations applied to the PDB |
| `minAvailable` | *intstr.IntOrString | Minimum number of pods that must be available |
| `maxUnavailable` | *intstr.IntOrString | Maximum number of pods that can be unavailable |
| `selector` | *metav1.LabelSelector | Label selector to identify pods |
| `unhealthyPodEvictionPolicy` | *string | Policy for evicting unhealthy pods |
| `currentHealthy` | int32 | Current number of healthy pods |
| `desiredHealthy` | int32 | Desired number of healthy pods |
| `expectedPods` | int32 | Total number of pods expected |
| `disruptionsAllowed` | int32 | Number of pods that can be disrupted |
| `conditions` | []metav1.Condition | Current conditions of the PDB |
| `clusterName` | string | Name of the Kubernetes cluster |
| `instanceHash` | string | Hash identifying the kubegraph instance |

## Neo4j Node Structure

```cypher
(:PodDisruptionBudget {
  name: "my-app-pdb",
  uid: "12345678-1234-1234-1234-123456789abc",
  namespace: "default",
  creationTimestamp: "2024-01-01T00:00:00Z",
  labels: {app: "my-app"},
  annotations: {description: "PDB for my application"},
  minAvailable: "2",
  maxUnavailable: "1",
  selector: {matchLabels: {app: "my-app"}},
  unhealthyPodEvictionPolicy: "IfHealthyBudget",
  currentHealthy: 3,
  desiredHealthy: 2,
  expectedPods: 3,
  disruptionsAllowed: 1,
  clusterName: "my-cluster",
  instanceHash: "abc123"
})
```

## Relationships

### Owner References

The handler creates `OWNED_BY` relationships to parent resources based on owner references:

```cypher
(:PodDisruptionBudget)-[:OWNED_BY]->(:ParentResource)
```

### Example Queries

#### List all PDBs in a namespace

```cypher
MATCH (pdb:PodDisruptionBudget {namespace: "default"})
RETURN pdb.name, pdb.minAvailable, pdb.maxUnavailable, pdb.currentHealthy, pdb.disruptionsAllowed
ORDER BY pdb.name
```

#### Find PDBs with specific selectors

```cypher
MATCH (pdb:PodDisruptionBudget)
WHERE pdb.selector.matchLabels.app = "my-app"
RETURN pdb.name, pdb.namespace, pdb.selector
```

#### Get PDB availability statistics

```cypher
MATCH (pdb:PodDisruptionBudget)
RETURN pdb.namespace,
       count(pdb) as pdbCount,
       avg(pdb.currentHealthy) as avgHealthy,
       avg(pdb.disruptionsAllowed) as avgDisruptionsAllowed
ORDER BY pdbCount DESC
```

#### Find PDBs with no disruptions allowed

```cypher
MATCH (pdb:PodDisruptionBudget)
WHERE pdb.disruptionsAllowed = 0
RETURN pdb.name, pdb.namespace, pdb.currentHealthy, pdb.desiredHealthy
```

#### Find PDBs with high availability requirements

```cypher
MATCH (pdb:PodDisruptionBudget)
WHERE pdb.minAvailable IS NOT NULL AND pdb.minAvailable > "50%"
RETURN pdb.name, pdb.namespace, pdb.minAvailable, pdb.currentHealthy
```

#### Get PDB hierarchy with owner references

```cypher
MATCH (pdb:PodDisruptionBudget)-[:OWNED_BY]->(owner)
WHERE pdb.name = "my-pdb"
RETURN pdb.name, owner.name, labels(owner)[0] as ownerType
```

#### Find PDBs with unhealthy pod eviction policies

```cypher
MATCH (pdb:PodDisruptionBudget)
WHERE pdb.unhealthyPodEvictionPolicy IS NOT NULL
RETURN pdb.name, pdb.namespace, pdb.unhealthyPodEvictionPolicy
```

#### Analyze PDB availability patterns

```cypher
MATCH (pdb:PodDisruptionBudget)
WHERE pdb.currentHealthy < pdb.desiredHealthy
RETURN pdb.name, pdb.namespace, 
       pdb.currentHealthy, pdb.desiredHealthy,
       pdb.desiredHealthy - pdb.currentHealthy as healthGap
```

#### Count PDBs per namespace

```cypher
MATCH (pdb:PodDisruptionBudget)
RETURN pdb.namespace, count(pdb) as pdbCount
ORDER BY pdbCount DESC
```

## Implementation Details

### Handler Registration

The PodDisruptionBudget handler is automatically registered in the Kubernetes client:

```go
handlers.NewPodDisruptionBudgetHandler(c.config)
```

### Event Handling

- **Create/Update**: Creates or updates the PDB node and establishes owner reference relationships
- **Delete**: Removes the PDB node and all its relationships from the graph

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

1. **Availability Management**: Monitor application availability during disruptions
2. **Maintenance Planning**: Plan cluster maintenance with minimal service impact
3. **Disaster Recovery**: Ensure critical applications remain available during node failures
4. **Capacity Planning**: Understand availability requirements across applications
5. **Compliance**: Meet availability SLAs and compliance requirements

## Availability Policies

The handler tracks various availability policies:
- **minAvailable**: Minimum number of pods that must be available
- **maxUnavailable**: Maximum number of pods that can be unavailable
- **unhealthyPodEvictionPolicy**: How to handle unhealthy pods during eviction

## Related Resources

- [Kubernetes PDB Documentation](https://kubernetes.io/docs/concepts/workloads/pods/disruptions/)
- [Neo4j Relationships Documentation](docs/neo4j_relationships.md)
- [Neo4j Ownership Queries](docs/neo4j_ownership_queries.md) 
