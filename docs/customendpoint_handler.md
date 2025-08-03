# CustomEndpoint Handler

The CustomEndpoint handler is responsible for monitoring and managing CustomEndpoint resources in Kubernetes clusters and storing their information in Neo4j.

## Overview

CustomEndpoint resources are part of the Neo4j Ingress Config system and are used to manage custom domain endpoints for Neo4j databases. They provide custom FQDNs (Fully Qualified Domain Names) that can be used to access specific Neo4j databases through custom domains.

## Properties Stored

The handler stores the following properties for each CustomEndpoint in Neo4j:

- `name`: The name of the CustomEndpoint resource
- `uid`: Unique identifier for the CustomEndpoint
- `namespace`: Kubernetes namespace where the CustomEndpoint is located
- `creationTimestamp`: When the CustomEndpoint was created
- `labels`: Kubernetes labels applied to the CustomEndpoint
- `annotations`: Kubernetes annotations applied to the CustomEndpoint
- `id`: Unique identifier for the custom endpoint
- `fqdn`: Fully Qualified Domain Name for the custom endpoint
- `dbid`: Database ID that this endpoint exposes
- `orchestra`: Orchestra identifier for the endpoint
- `clusterName`: Name of the Kubernetes cluster
- `instanceHash`: Instance hash for multi-instance deployments

## Relationships Created

The handler creates the following relationships in Neo4j:

1. **OWNED_BY**: Links CustomEndpoint to its owner resources
2. **EXPOSES**: Links CustomEndpoint to the Neo4jDatabase it exposes

### Ownership Relationships

The handler automatically creates `OWNED_BY` relationships based on the CustomEndpoint's `ownerReferences`. This typically includes:
- **Other controllers**: Any resource that creates CustomEndpoint resources

### Database Exposure Relationships

The handler creates `EXPOSES` relationships with Neo4jDatabase resources based on the `dbid` field. This provides a direct link between the CustomEndpoint and the database it's exposing.

## Usage

The CustomEndpoint handler is automatically registered when the application starts. It will:

1. Watch for CustomEndpoint resources in all namespaces
2. Process create/update events to store CustomEndpoint information
3. Process delete events to remove CustomEndpoint information
4. Create relationships with related Kubernetes resources
5. Establish ownership relationships based on owner references

## Example Neo4j Queries

### Basic CustomEndpoint Queries

```cypher
// Find all CustomEndpoint resources
MATCH (ce:CustomEndpoint) RETURN ce

// Find CustomEndpoint resources in a specific namespace
MATCH (ce:CustomEndpoint {namespace: "default"}) RETURN ce

// Find CustomEndpoint resources with specific labels
MATCH (ce:CustomEndpoint) WHERE ce.labels.app = "neo4j" RETURN ce
```

### Relationship Queries

```cypher
// Find CustomEndpoint resources and the databases they expose
MATCH (ce:CustomEndpoint)-[:EXPOSES]->(db:Neo4jDatabase) RETURN ce, db

// Find CustomEndpoint resources and their owners
MATCH (ce:CustomEndpoint)-[:OWNED_BY]->(owner) RETURN ce, owner

// Find databases with their custom endpoints
MATCH (db:Neo4jDatabase)<-[:EXPOSES]-(ce:CustomEndpoint) RETURN db, ce
```

### Domain and FQDN Queries

```cypher
// Find CustomEndpoint resources by FQDN pattern
MATCH (ce:CustomEndpoint) WHERE ce.fqdn CONTAINS "myapp" RETURN ce

// Find CustomEndpoint resources with specific domain patterns
MATCH (ce:CustomEndpoint) 
WHERE ce.fqdn CONTAINS ".endpoints.neo4j.io" 
RETURN ce.name, ce.fqdn

// Find all unique FQDNs
MATCH (ce:CustomEndpoint) 
RETURN DISTINCT ce.fqdn as domain, count(ce) as endpointCount
```

### Database Exposure Queries

```cypher
// Find databases with custom endpoints
MATCH (db:Neo4jDatabase)<-[:EXPOSES]-(ce:CustomEndpoint)
RETURN db.name, collect(ce.fqdn) as customDomains

// Find databases without custom endpoints
MATCH (db:Neo4jDatabase)
WHERE NOT EXISTS((db)<-[:EXPOSES]-(:CustomEndpoint))
RETURN db.name, db.namespace

// Find databases with multiple custom endpoints
MATCH (db:Neo4jDatabase)<-[:EXPOSES]-(ce:CustomEndpoint)
WITH db, collect(ce) as endpoints
WHERE size(endpoints) > 1
RETURN db.name, size(endpoints) as endpointCount
```

### Orchestra Queries

```cypher
// Find CustomEndpoint resources by orchestra
MATCH (ce:CustomEndpoint) WHERE ce.orchestra CONTAINS "orch-1234" RETURN ce

// Find all orchestras with their endpoints
MATCH (ce:CustomEndpoint)
RETURN ce.orchestra, collect(ce.fqdn) as endpoints, count(ce) as endpointCount
```

### Complex Endpoint Analysis

```cypher
// Find complete endpoint configuration for databases
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)<-[:EXPOSES]-(ce:CustomEndpoint)
OPTIONAL MATCH (db)<-[:PROTECTS]-(ipac:IPAccessControl)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
RETURN db.name,
       CASE WHEN ce IS NOT NULL THEN "Custom Domain" ELSE "Default Domain" END as endpointType,
       ce.fqdn as customDomain,
       ce.orchestra as orchestra,
       CASE WHEN ipac IS NOT NULL THEN "Protected" ELSE "Unprotected" END as ipProtection,
       cm.name as configMap
```

### Security and Access Analysis

```cypher
// Find databases with both custom endpoints and IP access controls
MATCH (db:Neo4jDatabase)<-[:EXPOSES]-(ce:CustomEndpoint)
MATCH (db)<-[:PROTECTS]-(ipac:IPAccessControl)
RETURN db.name, ce.fqdn, ipac.allowList as allowedIPs

// Find custom endpoints for unprotected databases
MATCH (db:Neo4jDatabase)<-[:EXPOSES]-(ce:CustomEndpoint)
WHERE NOT EXISTS((db)<-[:PROTECTS]-(:IPAccessControl))
RETURN db.name, ce.fqdn, "WARNING: No IP protection" as status
```

## Integration with Other Handlers

The CustomEndpoint handler works in conjunction with other handlers:

- **Neo4jDatabase Handler**: CustomEndpoint resources expose Neo4jDatabase resources
- **IPAccessControl Handler**: Custom endpoints may have associated IP access controls
- **Neo4jCluster Handler**: Clusters may have associated custom endpoints
- **Neo4jSingleInstance Handler**: Single instances may have associated custom endpoints

This creates a comprehensive endpoint graph showing which databases are exposed through custom domains and how they're configured.

## Endpoint Management Implications

The CustomEndpoint handler provides visibility into:

- **Domain Management**: Which databases have custom domain endpoints
- **Orchestra Configuration**: How endpoints are organized by orchestra
- **Database Exposure**: Which databases are accessible through custom domains
- **Endpoint Patterns**: Common FQDN patterns and naming conventions
- **Security Integration**: How custom endpoints work with IP access controls

This information is crucial for:
- **DNS Management**: Understanding custom domain configurations
- **Access Control**: Ensuring proper security for custom endpoints
- **Load Balancing**: Understanding endpoint distribution across orchestras
- **Compliance**: Tracking which databases are exposed through custom domains
- **Troubleshooting**: Identifying endpoint configuration issues

The CustomEndpoint handler helps you understand your complete Neo4j endpoint architecture and how custom domains are configured for database access. 
