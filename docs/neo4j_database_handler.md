# Neo4jDatabase Handler

## Overview

The Neo4jDatabase handler is responsible for processing Neo4j Database Custom Resources (CRs) and storing their information in Neo4j. This handler supports the `neo4j.io/v1` API group and manages `Neo4jDatabase` resources.

## Features

- **Complete CRD Support**: Handles all fields from the Neo4jDatabase CRD specification
- **Status Processing**: Extracts and stores comprehensive status information
- **Relationship Management**: Creates relationships with related Kubernetes resources
- **Rich Metadata**: Captures detailed configuration and operational data
- **Dynamic Ownership**: Creates ownership relationships based on deployment type (single instance vs cluster)

## Resource Information Captured

### Spec Fields
- `clusterId`: Unique cluster identifier
- `coreCount`: Number of core instances
- `coreDisk`: Disk configuration (size, virtual size)
- `coreMemory`: Memory configuration (size, packing efficiency)
- `cypherRoles`: Cypher roles for the database
- `databaseName`: Name of the database
- `dbid`: Database identifier
- `dnsName`: DNS name for the database
- `features`: List of enabled features
- `fineGrainedRBAC`: Fine-grained RBAC flag
- `multiZone`: Multi-zone deployment flag
- `neo4jMajorVersion`: Neo4j major version
- `primariesCount`: Number of primary instances
- `publicBoltPort`: Public Bolt port
- `zone`: Availability zone
- `additionalSettings`: Backup, logical restrictions, and plugins configuration
- `singleInstance`: Flag indicating if this is a single instance deployment

### Status Fields
- `phase`: Current phase of the database
- `conditions`: Status conditions
- `clusterStatuses`: Cluster status information
- `databaseCounts`: Database statistics (nodes, relationships)
- `showDatabasesStatus`: Database status information
- `showServersStatus`: Server status information
- `unifiedServerStatuses`: Unified server status information
- `raftStatuses`: Raft status information
- `neo4jPlugins`: Plugin information
- `systemDatabaseLeader`: System database leader
- `userDatabaseLeader`: User database leader
- `appliedRoles`: Applied roles information
- `cypherRoleNames`: Cypher role names
- `databaseSizes`: Database size information
- `diskUsageBytes`: Disk usage in bytes
- `hostClusterId`: Host cluster identifier
- `observedGeneration`: Observed generation
- `readyToMonitor`: Ready to monitor flag

## Relationships Created

The handler creates the following relationships in Neo4j:

1. **MANAGED_BY StatefulSet**: Links Neo4jDatabase to its managing StatefulSet
2. **USES ConfigMap**: Links Neo4jDatabase to its configuration ConfigMap
3. **OWNS Neo4jCluster**: Links Neo4jDatabase to the Neo4jCluster it owns (when `SingleInstance` is false)
4. **OWNS Neo4jSingleInstance**: Links Neo4jDatabase to the Neo4jSingleInstance it owns (when `SingleInstance` is true)

### Ownership Logic

The handler implements dynamic ownership relationships based on the `SingleInstance` field in the Neo4jDatabase spec:

- **When `SingleInstance` is `true`**: Creates an `OWNS` relationship to `Neo4jSingleInstance` using the `dbid` field
- **When `SingleInstance` is `false`**: Creates an `OWNS` relationship to `Neo4jCluster` using the `targetHostClusterId` field (with fallback to `hostClusterId` from status)

## Usage

The Neo4jDatabase handler is automatically registered when the application starts. It will:

1. Watch for Neo4jDatabase CRs in all namespaces
2. Process create/update events to store database information
3. Process delete events to remove database information
4. Create relationships with related Kubernetes resources
5. Establish ownership relationships based on deployment type

## Example Neo4j Queries

### Basic Neo4jDatabase Queries

```cypher
// Find all Neo4jDatabases
MATCH (db:Neo4jDatabase) RETURN db

// Find Neo4jDatabases with specific cluster
MATCH (db:Neo4jDatabase {clusterId: "my-cluster"}) RETURN db

// Find Neo4jDatabases and their StatefulSets
MATCH (db:Neo4jDatabase)-[:MANAGED_BY]->(ss:StatefulSet) RETURN db, ss

// Find Neo4jDatabases with high disk usage
MATCH (db:Neo4jDatabase) WHERE db.diskUsageBytes > 1000000000 RETURN db.name, db.diskUsageBytes
```

### Ownership Relationship Queries

```cypher
// Find all Neo4jDatabases and their ownership relationships
MATCH (db:Neo4jDatabase)-[r:OWNS]->(owner)
RETURN db.name as database_name, 
       type(r) as relationship_type, 
       labels(owner) as owner_type, 
       owner.name as owner_name

// Find single instance databases and their Neo4jSingleInstance owners
MATCH (db:Neo4jDatabase)-[:OWNS]->(si:Neo4jSingleInstance)
RETURN db.name as database_name, 
       db.dbid as database_id,
       si.name as single_instance_name

// Find clustered databases and their Neo4jCluster owners
MATCH (db:Neo4jDatabase)-[:OWNS]->(cluster:Neo4jCluster)
RETURN db.name as database_name, 
       db.targetHostClusterId as cluster_id,
       cluster.name as cluster_name

// Find databases by deployment type
MATCH (db:Neo4jDatabase)
WHERE db.singleInstance = true
RETURN db.name as single_instance_database

MATCH (db:Neo4jDatabase)
WHERE db.singleInstance = false
RETURN db.name as clustered_database
```

### Complex Relationship Queries

```cypher
// Find complete resource hierarchy for a database
MATCH (db:Neo4jDatabase {name: "my-database"})-[:OWNS]->(owner)
MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
RETURN db.name as database,
       labels(owner) as owner_type,
       owner.name as owner_name,
       ss.name as statefulset,
       cm.name as configmap,
       collect(pod.name) as pods

// Find all resources owned by a specific Neo4jCluster
MATCH (cluster:Neo4jCluster {name: "my-cluster"})<-[:OWNS]-(db:Neo4jDatabase)
MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
MATCH (db)-[:USES]->(cm:ConfigMap)
RETURN cluster.name as cluster,
       collect(db.name) as databases,
       collect(ss.name) as statefulsets,
       collect(cm.name) as configmaps

// Find resource usage patterns
MATCH (db:Neo4jDatabase)-[:USES]->(cm:ConfigMap)
WITH cm.name as configmap, count(db) as usage_count
WHERE usage_count > 1
RETURN configmap, usage_count
ORDER BY usage_count DESC
```

### Performance and Monitoring Queries

```cypher
// Find databases with high resource usage
MATCH (db:Neo4jDatabase)
WHERE db.diskUsageBytes > 5000000000 OR db.currentNodes > 1000000
RETURN db.name as database,
       db.diskUsageBytes as disk_usage,
       db.currentNodes as node_count,
       db.currentRelationships as relationship_count

// Find databases by status
MATCH (db:Neo4jDatabase)
WHERE db.phase = "Running"
RETURN db.name as database,
       db.phase as status,
       db.conditions as conditions

// Find databases with recent activity
MATCH (db:Neo4jDatabase)
WHERE db.lastActivity > datetime() - duration({days: 7})
RETURN db.name as database,
       db.lastActivity as last_activity
ORDER BY db.lastActivity DESC
```

## Configuration

The handler uses the following configuration from the main application:

- `clusterName`: Kubernetes cluster name
- `instanceHash`: Unique instance identifier for cleanup operations

## Error Handling

The handler includes comprehensive error handling:

- Type conversion errors for malformed resources
- Neo4j connection and query errors
- Relationship creation failures
- Graceful handling of missing optional fields
- Fallback logic for ownership relationship creation

## Dependencies

- `k8s.io/apimachinery/pkg/runtime/schema` for GVR definition
- `github.com/neo-technology/neo4j-cloud/libs/neo4j-database-clientset/apis/neo4j.io/v1` for Neo4jDatabase type definitions
- `kubegraph/pkg/neo4j` for Neo4j client operations 
