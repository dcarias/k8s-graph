# ReplicaSet Handler

The ReplicaSet handler is responsible for monitoring and managing ReplicaSet resources in Kubernetes clusters and storing their information in Neo4j.

## Overview

ReplicaSets are Kubernetes resources that ensure a specified number of pod replicas are running at any given time. They are typically managed by Deployments and provide the underlying mechanism for scaling and updating applications.

## Properties Stored

The handler stores the following properties for each ReplicaSet in Neo4j:

- `name`: The name of the ReplicaSet
- `uid`: Unique identifier for the ReplicaSet
- `namespace`: Kubernetes namespace where the ReplicaSet is located
- `creationTimestamp`: When the ReplicaSet was created
- `labels`: Kubernetes labels applied to the ReplicaSet
- `annotations`: Kubernetes annotations applied to the ReplicaSet
- `replicas`: Desired number of replicas
- `availableReplicas`: Number of available replicas
- `readyReplicas`: Number of ready replicas
- `selector`: Label selector used to identify managed pods
- `clusterName`: Name of the Kubernetes cluster
- `instanceHash`: Instance hash for multi-instance deployments

## Relationships Created

The handler creates the following relationships in Neo4j:

1. **OWNED_BY**: Links ReplicaSet to its owner resources (e.g., Deployments)
2. **MANAGES**: Links ReplicaSet to the Pods it manages

### Ownership Relationships

The handler automatically creates `OWNED_BY` relationships based on the ReplicaSet's `ownerReferences`. This typically includes:
- **Deployment**: Most ReplicaSets are owned by Deployments
- **StatefulSet**: Some ReplicaSets may be owned by StatefulSets
- **Other controllers**: Any other resource that creates ReplicaSets

### Pod Management Relationships

The handler creates `MANAGES` relationships with all Pods that match the ReplicaSet's label selector. This provides a direct link between the ReplicaSet and the Pods it's responsible for managing.

## Usage

The ReplicaSet handler is automatically registered when the application starts. It will:

1. Watch for ReplicaSet resources in all namespaces
2. Process create/update events to store ReplicaSet information
3. Process delete events to remove ReplicaSet information
4. Create relationships with related Kubernetes resources
5. Establish ownership relationships based on owner references

## Example Neo4j Queries

### Basic ReplicaSet Queries

```cypher
// Find all ReplicaSets
MATCH (rs:ReplicaSet) RETURN rs

// Find ReplicaSets in a specific namespace
MATCH (rs:ReplicaSet {namespace: "default"}) RETURN rs

// Find ReplicaSets with specific labels
MATCH (rs:ReplicaSet) WHERE rs.labels.app = "myapp" RETURN rs
```

### Relationship Queries

```cypher
// Find ReplicaSets and their owners
MATCH (rs:ReplicaSet)-[:OWNED_BY]->(owner) RETURN rs, owner

// Find ReplicaSets and the pods they manage
MATCH (rs:ReplicaSet)-[:MANAGES]->(pod:Pod) RETURN rs, pod

// Find Deployment -> ReplicaSet -> Pod chain
MATCH (deployment:Deployment)-[:OWNS]->(rs:ReplicaSet)-[:MANAGES]->(pod:Pod)
RETURN deployment, rs, pod
```

### Status Queries

```cypher
// Find ReplicaSets with scaling issues
MATCH (rs:ReplicaSet) 
WHERE rs.availableReplicas < rs.replicas 
RETURN rs.name, rs.availableReplicas, rs.replicas

// Find ReplicaSets with no ready replicas
MATCH (rs:ReplicaSet) 
WHERE rs.readyReplicas = 0 
RETURN rs.name, rs.namespace
```

## Integration with Other Handlers

The ReplicaSet handler works in conjunction with other handlers:

- **Deployment Handler**: ReplicaSets are typically owned by Deployments
- **Pod Handler**: ReplicaSets manage Pods
- **Node Handler**: Pods managed by ReplicaSets run on Nodes

This creates a comprehensive graph of Kubernetes resources and their relationships. 
