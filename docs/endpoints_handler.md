# Endpoints Handler

## Overview

The Endpoints Handler tracks Kubernetes Endpoints resources in Neo4j. Endpoints resources define the network endpoints that a service can route traffic to, typically representing the IP addresses and ports of pods.

## Resource Information

- **API Group**: `core` (v1)
- **Version**: `v1`
- **Resource**: `endpoints`
- **Namespaced**: Yes
- **Neo4j Label**: `Endpoints`

## Properties Stored

### Core Properties
- `name`: The name of the Endpoints resource
- `uid`: Unique identifier for the Endpoints
- `namespace`: The namespace where the Endpoints is located
- `labels`: Kubernetes labels
- `annotations`: Kubernetes annotations
- `clusterName`: The cluster where this resource exists

### Endpoints-Specific Properties
- `subsets`: Array of endpoint subsets containing:
  - `addresses`: Array of endpoint addresses (IP addresses of pods)
  - `ports`: Array of endpoint ports (ports that pods are listening on)

## Relationships

### PROVIDES_ENDPOINTS_FOR
- **From**: Endpoints
- **To**: Service
- **Description**: Links the Endpoints to the Service it provides endpoints for (same name)

## Example Cypher Queries

### Find all Endpoints resources
```cypher
MATCH (e:Endpoints) 
RETURN e.name, e.namespace, size(e.subsets) as subsetCount
```

### Find Endpoints with their associated services
```cypher
MATCH (e:Endpoints)-[:PROVIDES_ENDPOINTS_FOR]->(s:Service)
RETURN e.name, e.namespace, s.name, s.namespace
```

### Find Endpoints with multiple subsets
```cypher
MATCH (e:Endpoints)
WHERE size(e.subsets) > 1
RETURN e.name, e.namespace, size(e.subsets) as subsetCount
```

### Find Endpoints by namespace
```cypher
MATCH (e:Endpoints {namespace: 'default'})
RETURN e.name, size(e.subsets) as subsetCount
```

### Find Endpoints with specific port
```cypher
MATCH (e:Endpoints)
WHERE ANY(subset IN e.subsets WHERE ANY(port IN subset.ports WHERE port.port = 8080))
RETURN e.name, e.namespace, port.port, port.protocol
```

### Find Endpoints with no subsets (empty endpoints)
```cypher
MATCH (e:Endpoints)
WHERE e.subsets IS NULL OR size(e.subsets) = 0
RETURN e.name, e.namespace
```

### Find Endpoints by cluster
```cypher
MATCH (e:Endpoints {clusterName: 'my-cluster'})
RETURN e.name, e.namespace, size(e.subsets) as subsetCount
```

### Find services without endpoints
```cypher
MATCH (s:Service)
WHERE NOT EXISTS((e:Endpoints)-[:PROVIDES_ENDPOINTS_FOR]->(s))
RETURN s.name, s.namespace
```

### Find endpoints with specific protocol
```cypher
MATCH (e:Endpoints)
WHERE ANY(subset IN e.subsets WHERE ANY(port IN subset.ports WHERE port.protocol = 'TCP'))
RETURN e.name, e.namespace, collect(DISTINCT port.protocol) as protocols
```

### Find endpoints with multiple addresses
```cypher
MATCH (e:Endpoints)
WHERE ANY(subset IN e.subsets WHERE size(subset.addresses) > 1)
RETURN e.name, e.namespace, 
       [subset IN e.subsets | size(subset.addresses)] as addressCounts
```

## Related Handlers

- **Service Handler**: Endpoints provide network endpoints for services
- **Pod Handler**: Endpoints represent the IP addresses and ports of pods
- **Namespace Handler**: Endpoints resources are namespaced

## Use Cases

- **Service Discovery**: Understanding which pods are backing a service
- **Load Balancing**: Identifying the available endpoints for traffic distribution
- **Network Troubleshooting**: Diagnosing connectivity issues between services and pods
- **Capacity Planning**: Understanding the number of endpoints available for services
- **Health Monitoring**: Identifying services with no available endpoints
- **Service Mesh Integration**: Understanding endpoint distribution for service mesh routing

## Notes

- Endpoints resources are automatically created and managed by Kubernetes for each Service
- The name of an Endpoints resource typically matches the name of its associated Service
- Endpoints can have multiple subsets, each containing addresses and ports
- Empty endpoints (no subsets) indicate that no pods are currently backing the service
- Endpoints are updated automatically when pods are created, deleted, or their labels change
- The relationship to Service is based on matching names in the same namespace 
