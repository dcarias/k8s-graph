# IPAccessControl Handler

The IPAccessControl handler is responsible for monitoring and managing IPAccessControl resources in Kubernetes clusters and storing their information in Neo4j.

## Overview

IPAccessControl resources are part of the Neo4j Ingress Config system and are used to manage IP-based access control for Neo4j databases. They define which IP addresses or CIDR blocks are allowed to access specific Neo4j databases.

## Properties Stored

The handler stores the following properties for each IPAccessControl in Neo4j:

- `name`: The name of the IPAccessControl resource
- `uid`: Unique identifier for the IPAccessControl
- `namespace`: Kubernetes namespace where the IPAccessControl is located
- `creationTimestamp`: When the IPAccessControl was created
- `labels`: Kubernetes labels applied to the IPAccessControl
- `annotations`: Kubernetes annotations applied to the IPAccessControl
- `dbid`: Database ID that this access control protects
- `tenantId`: Tenant ID associated with this access control
- `filteringDisabled`: Whether IP filtering is disabled
- `allowList`: List of allowed IP addresses/CIDR blocks
- `errorMessage`: Error message if there are issues
- `ipAccessControlStatus`: Current status of the access control (Active, Pending, Failed, Expired, Suspended)
- `clusterName`: Name of the Kubernetes cluster
- `instanceHash`: Instance hash for multi-instance deployments

## Relationships Created

The handler creates the following relationships in Neo4j:

1. **OWNED_BY**: Links IPAccessControl to its owner resources
2. **PROTECTS**: Links IPAccessControl to the Neo4jDatabase it protects

### Ownership Relationships

The handler automatically creates `OWNED_BY` relationships based on the IPAccessControl's `ownerReferences`. This typically includes:
- **Other controllers**: Any resource that creates IPAccessControl resources

### Database Protection Relationships

The handler creates `PROTECTS` relationships with Neo4jDatabase resources based on the `dbid` field. This provides a direct link between the IPAccessControl and the database it's protecting.

## Usage

The IPAccessControl handler is automatically registered when the application starts. It will:

1. Watch for IPAccessControl resources in all namespaces
2. Process create/update events to store IPAccessControl information
3. Process delete events to remove IPAccessControl information
4. Create relationships with related Kubernetes resources
5. Establish ownership relationships based on owner references

## Example Neo4j Queries

### Basic IPAccessControl Queries

```cypher
// Find all IPAccessControl resources
MATCH (ipac:IPAccessControl) RETURN ipac

// Find IPAccessControl resources in a specific namespace
MATCH (ipac:IPAccessControl {namespace: "default"}) RETURN ipac

// Find IPAccessControl resources with specific labels
MATCH (ipac:IPAccessControl) WHERE ipac.labels.app = "neo4j" RETURN ipac
```

### Relationship Queries

```cypher
// Find IPAccessControl resources and the databases they protect
MATCH (ipac:IPAccessControl)-[:PROTECTS]->(db:Neo4jDatabase) RETURN ipac, db

// Find IPAccessControl resources and their owners
MATCH (ipac:IPAccessControl)-[:OWNED_BY]->(owner) RETURN ipac, owner

// Find databases with their IP access controls
MATCH (db:Neo4jDatabase)<-[:PROTECTS]-(ipac:IPAccessControl) RETURN db, ipac
```

### Status Queries

```cypher
// Find active IPAccessControl resources
MATCH (ipac:IPAccessControl) WHERE ipac.ipAccessControlStatus = "Active" RETURN ipac

// Find failed IPAccessControl resources
MATCH (ipac:IPAccessControl) WHERE ipac.ipAccessControlStatus = "Failed" RETURN ipac

// Find IPAccessControl resources with errors
MATCH (ipac:IPAccessControl) WHERE ipac.errorMessage IS NOT NULL RETURN ipac.name, ipac.errorMessage

// Find disabled IP filtering
MATCH (ipac:IPAccessControl) WHERE ipac.filteringDisabled = true RETURN ipac
```

### Security Queries

```cypher
// Find databases without IP access controls
MATCH (db:Neo4jDatabase)
WHERE NOT EXISTS((db)<-[:PROTECTS]-(:IPAccessControl))
RETURN db.name, db.namespace

// Find IPAccessControl resources with specific CIDR blocks
MATCH (ipac:IPAccessControl)
WHERE ANY(cidr IN ipac.allowList WHERE cidr CONTAINS "192.168.")
RETURN ipac.name, ipac.allowList

// Find databases protected by multiple IP access controls
MATCH (db:Neo4jDatabase)<-[:PROTECTS]-(ipac:IPAccessControl)
WITH db, collect(ipac) as controls
WHERE size(controls) > 1
RETURN db.name, size(controls) as controlCount
```

### Tenant Queries

```cypher
// Find IPAccessControl resources by tenant
MATCH (ipac:IPAccessControl) WHERE ipac.tenantId = "tenant-123" RETURN ipac

// Find all tenants with their access controls
MATCH (ipac:IPAccessControl)
RETURN ipac.tenantId, collect(ipac.name) as accessControls, count(ipac) as controlCount
```

### Complex Security Analysis

```cypher
// Find databases with their complete security setup
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)<-[:PROTECTS]-(ipac:IPAccessControl)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(sts:StatefulSet)
RETURN db.name, 
       CASE WHEN ipac IS NOT NULL THEN "Protected" ELSE "Unprotected" END as ipProtection,
       CASE WHEN ipac.filteringDisabled = true THEN "Disabled" ELSE "Enabled" END as filteringStatus,
       ipac.allowList as allowedIPs,
       cm.name as configMap,
       sts.name as statefulSet
```

## Integration with Other Handlers

The IPAccessControl handler works in conjunction with other handlers:

- **Neo4jDatabase Handler**: IPAccessControl resources protect Neo4jDatabase resources
- **Neo4jCluster Handler**: Clusters may have associated IP access controls
- **Neo4jSingleInstance Handler**: Single instances may have associated IP access controls

This creates a comprehensive security graph showing which databases are protected by IP access controls and what IP ranges are allowed to access them.

## Security Implications

The IPAccessControl handler provides visibility into:

- **Database Security**: Which databases have IP-based access controls
- **Access Patterns**: What IP ranges are allowed to access specific databases
- **Security Gaps**: Databases without IP access controls
- **Configuration Issues**: Failed or suspended access controls
- **Tenant Isolation**: How different tenants' databases are protected

This information is crucial for security auditing, compliance reporting, and understanding the overall security posture of your Neo4j infrastructure. 
