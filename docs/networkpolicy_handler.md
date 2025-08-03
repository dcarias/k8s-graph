# NetworkPolicy Handler

## Overview

The NetworkPolicy Handler tracks Kubernetes NetworkPolicy resources in Neo4j. NetworkPolicy resources define network access rules that control how pods communicate with each other and other network endpoints.

## Resource Information

- **API Group**: `networking.k8s.io`
- **Version**: `v1`
- **Resource**: `networkpolicies`
- **Namespaced**: Yes
- **Neo4j Label**: `NetworkPolicy`

## Properties Stored

### Core Properties
- `name`: The name of the NetworkPolicy resource
- `uid`: Unique identifier for the NetworkPolicy
- `namespace`: The namespace where the NetworkPolicy is located
- `labels`: Kubernetes labels
- `annotations`: Kubernetes annotations
- `clusterName`: The cluster where this resource exists

### NetworkPolicy-Specific Properties
- `policyTypes`: Array of policy types (Ingress, Egress, or both)
- `ingress`: Array of ingress rules containing:
  - `ports`: Array of port configurations containing:
    - `protocol`: The protocol (TCP, UDP, SCTP)
    - `port`: The port number or name
    - `endPort`: The end port for port ranges
  - `from`: Array of source configurations containing:
    - `namespaceSelector`: Label selector for namespaces
    - `podSelector`: Label selector for pods
    - `ipBlock`: IP block configuration containing:
      - `cidr`: The CIDR block
      - `except`: Array of excluded CIDR blocks
- `egress`: Array of egress rules containing:
  - `ports`: Array of port configurations (same structure as ingress)
  - `to`: Array of destination configurations (same structure as ingress `from`)

## Relationships

NetworkPolicy resources don't create explicit relationships in Neo4j, but they can be queried in relation to other resources based on their selectors and rules.

## Example Cypher Queries

### Find all NetworkPolicy resources
```cypher
MATCH (np:NetworkPolicy) 
RETURN np.name, np.namespace, np.policyTypes
```

### Find NetworkPolicy resources by namespace
```cypher
MATCH (np:NetworkPolicy {namespace: 'default'})
RETURN np.name, np.policyTypes
```

### Find NetworkPolicy resources with both ingress and egress rules
```cypher
MATCH (np:NetworkPolicy)
WHERE 'Ingress' IN np.policyTypes AND 'Egress' IN np.policyTypes
RETURN np.name, np.namespace, np.policyTypes
```

### Find NetworkPolicy resources with ingress rules only
```cypher
MATCH (np:NetworkPolicy)
WHERE 'Ingress' IN np.policyTypes AND NOT ('Egress' IN np.policyTypes)
RETURN np.name, np.namespace
```

### Find NetworkPolicy resources with egress rules only
```cypher
MATCH (np:NetworkPolicy)
WHERE 'Egress' IN np.policyTypes AND NOT ('Ingress' IN np.policyTypes)
RETURN np.name, np.namespace
```

### Find NetworkPolicy resources with specific port
```cypher
MATCH (np:NetworkPolicy)
WHERE ANY(rule IN np.ingress WHERE ANY(port IN rule.ports WHERE port.port = 80))
RETURN np.name, np.namespace, port.port, port.protocol
```

### Find NetworkPolicy resources with IP block rules
```cypher
MATCH (np:NetworkPolicy)
WHERE ANY(rule IN np.ingress WHERE ANY(from IN rule.from WHERE from.ipBlock IS NOT NULL))
RETURN np.name, np.namespace, from.ipBlock.cidr
```

### Find NetworkPolicy resources by cluster
```cypher
MATCH (np:NetworkPolicy {clusterName: 'my-cluster'})
RETURN np.name, np.namespace, np.policyTypes
```

### Find NetworkPolicy resources with namespace selectors
```cypher
MATCH (np:NetworkPolicy)
WHERE ANY(rule IN np.ingress WHERE ANY(from IN rule.from WHERE from.namespaceSelector IS NOT NULL))
RETURN np.name, np.namespace, from.namespaceSelector
```

### Find NetworkPolicy resources with pod selectors
```cypher
MATCH (np:NetworkPolicy)
WHERE ANY(rule IN np.ingress WHERE ANY(from IN rule.from WHERE from.podSelector IS NOT NULL))
RETURN np.name, np.namespace, from.podSelector
```

### Find NetworkPolicy resources with complex rules
```cypher
MATCH (np:NetworkPolicy)
WHERE size(np.ingress) > 1 OR size(np.egress) > 1
RETURN np.name, np.namespace, 
       size(np.ingress) as ingressRuleCount,
       size(np.egress) as egressRuleCount
```

### Find NetworkPolicy resources with specific protocol
```cypher
MATCH (np:NetworkPolicy)
WHERE ANY(rule IN np.ingress WHERE ANY(port IN rule.ports WHERE port.protocol = 'TCP'))
RETURN np.name, np.namespace, collect(DISTINCT port.protocol) as protocols
```

### Find NetworkPolicy resources with port ranges
```cypher
MATCH (np:NetworkPolicy)
WHERE ANY(rule IN np.ingress WHERE ANY(port IN rule.ports WHERE port.endPort IS NOT NULL))
RETURN np.name, np.namespace, port.port, port.endPort
```

## Related Handlers

- **Pod Handler**: NetworkPolicy rules target pods based on selectors
- **Namespace Handler**: NetworkPolicy resources are namespaced and can reference other namespaces
- **Service Handler**: NetworkPolicy can affect service-to-pod communication

## Use Cases

- **Network Security**: Understanding network access controls and security policies
- **Compliance**: Auditing network policies for security compliance
- **Troubleshooting**: Diagnosing network connectivity issues
- **Policy Management**: Understanding which pods are affected by network policies
- **Security Analysis**: Identifying pods with specific network access patterns
- **Network Architecture**: Mapping network security boundaries and access patterns

## Notes

- NetworkPolicy resources are namespaced and only affect pods in the same namespace
- Pod selectors determine which pods the policy applies to
- Namespace selectors can reference pods in other namespaces
- IP blocks can be used to allow/deny traffic from specific CIDR ranges
- Port rules can specify individual ports or port ranges
- Policy types determine whether ingress, egress, or both are controlled
- NetworkPolicy resources require a network plugin that supports them (like Calico, Cilium, etc.)
- Default policies (allow all) apply when no NetworkPolicy matches a pod 
