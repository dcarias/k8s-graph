# Ingress Handler

## Overview

The Ingress Handler tracks Kubernetes Ingress resources in Neo4j. Ingress resources define rules for routing external HTTP/HTTPS traffic to services within the cluster.

## Resource Information

- **API Group**: `networking.k8s.io`
- **Version**: `v1`
- **Resource**: `ingresses`
- **Namespaced**: Yes
- **Neo4j Label**: `Ingress`

## Properties Stored

### Core Properties
- `name`: The name of the Ingress resource
- `uid`: Unique identifier for the Ingress
- `namespace`: The namespace where the Ingress is located
- `ingressClassName`: The Ingress class name (optional)
- `labels`: Kubernetes labels
- `annotations`: Kubernetes annotations
- `clusterName`: The cluster where this resource exists

### Ingress-Specific Properties
- `rules`: Array of ingress rules containing:
  - `host`: The hostname for the rule
  - `paths`: Array of path configurations containing:
    - `path`: The URL path
    - `pathType`: The type of path matching (Exact, Prefix, ImplementationSpecific)
    - `serviceName`: The target service name
    - `servicePort`: The target service port
- `tls`: Array of TLS configurations containing:
  - `secretName`: The name of the TLS secret
  - `hosts`: Array of hostnames for TLS
- `loadBalancerStatus`: Load balancer status information

## Relationships

### ROUTES_TO
- **From**: Ingress
- **To**: Service
- **Description**: Indicates which services the Ingress routes traffic to

## Example Cypher Queries

### Find all Ingress resources
```cypher
MATCH (i:Ingress) 
RETURN i.name, i.namespace, i.ingressClassName
```

### Find Ingress resources with their target services
```cypher
MATCH (i:Ingress)-[:ROUTES_TO]->(s:Service)
RETURN i.name, i.namespace, s.name, s.namespace
```

### Find Ingress resources with TLS configuration
```cypher
MATCH (i:Ingress)
WHERE i.tls IS NOT NULL AND size(i.tls) > 0
RETURN i.name, i.namespace, i.tls
```

### Find Ingress resources by hostname
```cypher
MATCH (i:Ingress)
WHERE ANY(rule IN i.rules WHERE rule.host = 'example.com')
RETURN i.name, i.namespace, rule.host
```

### Find Ingress resources with specific path
```cypher
MATCH (i:Ingress)
WHERE ANY(rule IN i.rules WHERE ANY(path IN rule.paths WHERE path.path = '/api'))
RETURN i.name, i.namespace, path.path
```

### Find Ingress resources by namespace
```cypher
MATCH (i:Ingress {namespace: 'default'})
RETURN i.name, i.ingressClassName
```

### Find Ingress resources with load balancer status
```cypher
MATCH (i:Ingress)
WHERE i.loadBalancerStatus IS NOT NULL
RETURN i.name, i.namespace, i.loadBalancerStatus
```

### Find Ingress resources with multiple paths
```cypher
MATCH (i:Ingress)
WITH i, [rule IN i.rules WHERE size(rule.paths) > 1] as multiPathRules
WHERE size(multiPathRules) > 0
RETURN i.name, i.namespace, size(multiPathRules[0].paths) as pathCount
```

### Find Ingress resources by cluster
```cypher
MATCH (i:Ingress {clusterName: 'my-cluster'})
RETURN i.name, i.namespace
```

## Related Handlers

- **Service Handler**: Ingress resources route traffic to services
- **Secret Handler**: TLS configurations reference secrets
- **Namespace Handler**: Ingress resources are namespaced

## Use Cases

- **Traffic Routing**: Understanding how external traffic is routed to internal services
- **TLS Configuration**: Identifying which Ingress resources use TLS certificates
- **Service Discovery**: Finding which services are exposed externally
- **Load Balancer Management**: Monitoring load balancer status and configuration
- **Security Analysis**: Understanding external access patterns and TLS usage
- **Network Architecture**: Mapping the external-to-internal traffic flow

## Notes

- Ingress resources are namespaced and can only route traffic to services in the same namespace
- TLS configurations reference Kubernetes secrets for certificate management
- Load balancer status provides information about external IP addresses or hostnames
- Ingress rules can have multiple paths and target different services
- The Ingress class determines which Ingress controller handles the resource 
