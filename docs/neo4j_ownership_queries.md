# Neo4j Ownership Relationships and Queries

This document provides comprehensive information about the dynamic ownership relationships created by the Neo4j handlers and includes practical Cypher queries for analyzing these relationships.

## Overview

The Neo4j handlers implement dynamic ownership relationships that automatically adapt based on the deployment type of Neo4j databases. This creates a flexible and accurate representation of resource ownership in the graph database.

## Ownership Logic

### Neo4jDatabase Ownership

The Neo4jDatabase handler implements intelligent ownership relationships based on the `SingleInstance` field:

- **Single Instance Databases** (`SingleInstance: true`): Creates `OWNS` relationship to `Neo4jSingleInstance`
- **Clustered Databases** (`SingleInstance: false`): Creates `OWNS` relationship to `Neo4jCluster`

### Ownership Direction

The ownership relationship is unidirectional - only Neo4jDatabase owns Neo4jCluster or Neo4jSingleInstance:

- **Neo4jDatabase → Neo4jCluster**: Database owns the cluster it runs on
- **Neo4jDatabase → Neo4jSingleInstance**: Database owns the single instance it runs on

Neo4jCluster and Neo4jSingleInstance do not own Neo4jDatabase. They are the infrastructure resources that host the databases.

### Relationship Creation Details

```go
// From Neo4jDatabase handler
if neo4jDB.Spec.SingleInstance {
    // Single instance: OWNS → Neo4jSingleInstance
    if err := neo4jClient.CreateRelationship(ctx, "Neo4jDatabase", "uid", string(neo4jDB.UID), "OWNS", "Neo4jSingleInstance", "dbid", neo4jDB.Spec.DbId.String()); err != nil {
        return fmt.Errorf("failed to create relationship between Neo4jDatabase %s and Neo4jSingleInstance %s: %w", neo4jDB.Name, neo4jDB.Spec.DbId.String(), err)
    }
} else {
    // Clustered: OWNS → Neo4jCluster
    clusterId := neo4jDB.Spec.TargetHostClusterId
    if clusterId == "" {
        // Use hostClusterId as fallback if targetHostClusterId is empty
        clusterId = neo4jDB.Status.HostClusterId
    }
    if clusterId != "" {
        if err := neo4jClient.CreateRelationship(ctx, "Neo4jDatabase", "uid", string(neo4jDB.UID), "OWNS", "Neo4jCluster", "clusterId", clusterId); err != nil {
            return fmt.Errorf("failed to create relationship between Neo4jDatabase %s and Neo4jCluster %s: %w", neo4jDB.Name, clusterId, err)
        }
    }
}
```

## Cypher Queries

### Basic Ownership Queries

#### 1. Find All Databases and Their Ownership Type

```cypher
MATCH (db:Neo4jDatabase)-[r:OWNS]->(owner)
RETURN db.name as database_name, 
       type(r) as relationship_type, 
       labels(owner) as owner_type, 
       owner.name as owner_name,
       db.singleInstance as is_single_instance,
       db.phase as status
ORDER BY db.name
```

**Returns:**
- `database_name`: Name of the Neo4j database
- `relationship_type`: Type of relationship (OWNS)
- `owner_type`: Type of owner (Neo4jCluster or Neo4jSingleInstance)
- `owner_name`: Name of the owner
- `is_single_instance`: Boolean indicating single instance deployment
- `status`: Current phase of the database

#### 2. Count Databases by Ownership Type

```cypher
MATCH (db:Neo4jDatabase)-[:OWNS]->(owner)
WITH labels(owner) as owner_type, count(db) as count
RETURN owner_type[0] as owner_type, count
ORDER BY count DESC
```

**Returns:**
- `owner_type`: Type of owner (Neo4jCluster or Neo4jSingleInstance)
- `count`: Number of databases owned by this type

#### 3. Find Databases Without Ownership Relationships

```cypher
MATCH (db:Neo4jDatabase)
WHERE NOT (db)-[:OWNS]->()
RETURN db.name as database_name,
       db.singleInstance as single_instance,
       db.phase as status
```

**Returns:**
- `database_name`: Databases without ownership relationships
- `single_instance`: Single instance flag
- `status`: Current status

### Single Instance Database Queries

#### 4. Find All Single Instance Databases

```cypher
MATCH (db:Neo4jDatabase)-[:OWNS]->(si:Neo4jSingleInstance)
RETURN db.name as database_name, 
       db.dbid as database_id,
       si.name as single_instance_name,
       db.phase as status,
       db.coreCount as core_count,
       db.coreMemorySize as memory_size
ORDER BY db.name
```

**Returns:**
- `database_name`: Name of the database
- `database_id`: Database ID
- `single_instance_name`: Name of the single instance owner
- `status`: Current phase
- `core_count`: Number of cores
- `memory_size`: Memory allocation

#### 5. Single Instance Database Performance

```cypher
MATCH (db:Neo4jDatabase)-[:OWNS]->(si:Neo4jSingleInstance)
WHERE db.diskUsageBytes > 0 OR db.currentNodes > 0
RETURN db.name as database_name,
       si.name as single_instance_name,
       db.diskUsageBytes as disk_usage_bytes,
       db.currentNodes as node_count,
       db.currentRelationships as relationship_count,
       db.lastActivity as last_activity
ORDER BY db.diskUsageBytes DESC
```

**Returns:**
- `database_name`: Database name
- `single_instance_name`: Single instance name
- `disk_usage_bytes`: Current disk usage
- `node_count`: Number of nodes
- `relationship_count`: Number of relationships
- `last_activity`: Last activity timestamp

### Clustered Database Queries

#### 6. Find All Clustered Databases

```cypher
MATCH (db:Neo4jDatabase)-[:OWNS]->(cluster:Neo4jCluster)
RETURN db.name as database_name, 
       db.targetHostClusterId as cluster_id,
       cluster.name as cluster_name,
       db.coreCount as core_count,
       db.primariesCount as primaries_count,
       db.phase as status
ORDER BY db.name
```

**Returns:**
- `database_name`: Name of the database
- `cluster_id`: Target host cluster ID
- `cluster_name`: Name of the cluster
- `core_count`: Number of core instances
- `primaries_count`: Number of primary instances
- `status`: Current phase

#### 7. Cluster Resource Usage Analysis

```cypher
MATCH (cluster:Neo4jCluster)<-[:OWNS]-(db:Neo4jDatabase)
WITH cluster, 
     collect(db.name) as databases,
     sum(db.coreCount) as total_cores,
     sum(db.primariesCount) as total_primaries,
     sum(db.diskUsageBytes) as total_disk_usage
RETURN cluster.name as cluster_name,
       databases,
       total_cores,
       total_primaries,
       total_disk_usage,
       size(databases) as database_count
ORDER BY total_cores DESC
```

**Returns:**
- `cluster_name`: Name of the cluster
- `databases`: List of databases owned by the cluster
- `total_cores`: Total core count across all databases
- `total_primaries`: Total primary count across all databases
- `total_disk_usage`: Total disk usage across all databases
- `database_count`: Number of databases in the cluster

### Complex Relationship Queries

#### 8. Complete Resource Hierarchy for a Database

```cypher
MATCH (db:Neo4jDatabase {name: "my-database"})-[:OWNS]->(owner)
MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (pod)-[:SCHEDULED_ON]->(node:Node)
RETURN db.name as database,
       labels(owner) as owner_type,
       owner.name as owner_name,
       ss.name as statefulset,
       cm.name as configmap,
       collect(DISTINCT pod.name) as pods,
       collect(DISTINCT node.name) as nodes
```

**Returns:**
- `database`: Database name
- `owner_type`: Type of owner
- `owner_name`: Name of the owner
- `statefulset`: StatefulSet name
- `configmap`: ConfigMap name
- `pods`: List of pods
- `nodes`: List of nodes

#### 9. Resource Usage Patterns

```cypher
MATCH (db:Neo4jDatabase)-[:USES]->(cm:ConfigMap)
WITH cm.name as configmap, count(db) as usage_count
WHERE usage_count > 1
RETURN configmap, usage_count
ORDER BY usage_count DESC
```

**Returns:**
- `configmap`: ConfigMap name
- `usage_count`: Number of databases using this ConfigMap

#### 10. Cross-Cluster Analysis

```cypher
MATCH (cluster:Neo4jCluster)<-[:OWNS]-(db:Neo4jDatabase)
MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
WITH cluster, 
     collect(DISTINCT ss.name) as statefulsets,
     collect(DISTINCT db.name) as databases
RETURN cluster.name as cluster_name,
       databases,
       statefulsets,
       size(databases) as database_count,
       size(statefulsets) as statefulset_count
ORDER BY database_count DESC
```

**Returns:**
- `cluster_name`: Cluster name
- `databases`: List of databases
- `statefulsets`: List of StatefulSets
- `database_count`: Number of databases
- `statefulset_count`: Number of StatefulSets

### Performance and Monitoring Queries

#### 11. High Resource Usage Databases

```cypher
MATCH (db:Neo4jDatabase)
WHERE db.diskUsageBytes > 5000000000 OR db.currentNodes > 1000000
OPTIONAL MATCH (db)-[:OWNS]->(owner)
RETURN db.name as database,
       labels(owner) as owner_type,
       owner.name as owner_name,
       db.diskUsageBytes as disk_usage_bytes,
       db.currentNodes as node_count,
       db.currentRelationships as relationship_count,
       db.phase as status
ORDER BY db.diskUsageBytes DESC
```

**Returns:**
- `database`: Database name
- `owner_type`: Type of owner
- `owner_name`: Name of owner
- `disk_usage_bytes`: Disk usage
- `node_count`: Node count
- `relationship_count`: Relationship count
- `status`: Current status

#### 12. Database Status by Ownership Type

```cypher
MATCH (db:Neo4jDatabase)-[:OWNS]->(owner)
WITH labels(owner) as owner_type, db.phase as phase, count(db) as count
RETURN owner_type[0] as owner_type, phase, count
ORDER BY owner_type, phase
```

**Returns:**
- `owner_type`: Type of owner
- `phase`: Database phase
- `count`: Number of databases in this phase

#### 13. Recent Activity Analysis

```cypher
MATCH (db:Neo4jDatabase)
WHERE db.lastActivity > datetime() - duration({days: 7})
OPTIONAL MATCH (db)-[:OWNS]->(owner)
RETURN db.name as database,
       labels(owner) as owner_type,
       owner.name as owner_name,
       db.lastActivity as last_activity,
       db.phase as status
ORDER BY db.lastActivity DESC
```

**Returns:**
- `database`: Database name
- `owner_type`: Type of owner
- `owner_name`: Name of owner
- `last_activity`: Last activity timestamp
- `status`: Current status

### Troubleshooting Queries

#### 14. Find Orphaned Resources

```cypher
// Find StatefulSets without database ownership
MATCH (ss:StatefulSet)
WHERE ss.name CONTAINS 'neo4j' AND NOT (ss)<-[:MANAGED_BY]-()
RETURN ss.name as statefulset_name

// Find ConfigMaps without database usage
MATCH (cm:ConfigMap)
WHERE cm.name CONTAINS 'neo4j' AND NOT (cm)<-[:USES]-()
RETURN cm.name as configmap_name
```

#### 15. Relationship Validation

```cypher
// Validate ownership relationships
MATCH (db:Neo4jDatabase)
WHERE db.singleInstance = true AND NOT (db)-[:OWNS]->(:Neo4jSingleInstance)
RETURN db.name as missing_single_instance_relationship

MATCH (db:Neo4jDatabase)
WHERE db.singleInstance = false AND NOT (db)-[:OWNS]->(:Neo4jCluster)
RETURN db.name as missing_cluster_relationship
```

### Infrastructure Resource Queries

#### 16. Find All Infrastructure Resources

```cypher
// Find all Neo4jClusters and Neo4jSingleInstances
MATCH (cluster:Neo4jCluster)
RETURN cluster.name as name,
       'Neo4jCluster' as type,
       cluster.phase as phase
UNION ALL
MATCH (si:Neo4jSingleInstance)
RETURN si.name as name,
       'Neo4jSingleInstance' as type,
       si.phase as phase
ORDER BY type, name
```

**Returns:**
- `name`: Name of the infrastructure resource
- `type`: Type of resource (Neo4jCluster or Neo4jSingleInstance)
- `phase`: Current phase of the resource

#### 17. Find Infrastructure Resources Without Databases

```cypher
// Find clusters and single instances that don't have any databases owning them
MATCH (infra)
WHERE (infra:Neo4jCluster OR infra:Neo4jSingleInstance)
AND NOT (infra)<-[:OWNS]-()
RETURN infra.name as resource_name,
       labels(infra) as resource_type,
       infra.phase as phase
ORDER BY resource_type, resource_name
```

**Returns:**
- `resource_name`: Name of the infrastructure resource
- `resource_type`: Type of resource
- `phase`: Current phase

#### 18. Find Databases by Infrastructure Resource

```cypher
// Find all databases that own a specific cluster
MATCH (cluster:Neo4jCluster {name: "my-cluster"})<-[:OWNS]-(db:Neo4jDatabase)
RETURN cluster.name as cluster_name,
       collect(db.name) as owned_databases,
       collect(db.phase) as database_phases,
       count(db) as database_count

// Find all databases that own a specific single instance
MATCH (si:Neo4jSingleInstance {name: "my-single-instance"})<-[:OWNS]-(db:Neo4jDatabase)
RETURN si.name as single_instance_name,
       collect(db.name) as owned_databases,
       collect(db.phase) as database_phases,
       count(db) as database_count
```

**Returns:**
- `cluster_name`/`single_instance_name`: Name of the infrastructure resource
- `owned_databases`: List of databases that own this resource
- `database_phases`: List of database phases
- `database_count`: Number of databases

#### 19. Infrastructure Resource Utilization

```cypher
// Show infrastructure resource utilization
MATCH (infra)
WHERE (infra:Neo4jCluster OR infra:Neo4jSingleInstance)
OPTIONAL MATCH (infra)<-[:OWNS]-(db:Neo4jDatabase)
WITH infra, collect(db) as databases
RETURN infra.name as resource_name,
       labels(infra) as resource_type,
       infra.phase as phase,
       size(databases) as database_count,
       collect(db.name) as database_names
ORDER BY resource_type, database_count DESC
```

**Returns:**
- `resource_name`: Name of the infrastructure resource
- `resource_type`: Type of resource
- `phase`: Current phase
- `database_count`: Number of databases using this resource
- `database_names`: List of database names

#### 20. Complete Infrastructure Graph

```cypher
// Show complete infrastructure graph for a specific database
MATCH (db:Neo4jDatabase {name: "my-database"})
OPTIONAL MATCH (db)-[:OWNS]->(infra)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (infra)-[:OWNS]->(infraSS:StatefulSet)
OPTIONAL MATCH (infra)-[:USES]->(infraCM:ConfigMap)
RETURN db.name as database,
       db.singleInstance as single_instance,
       labels(infra) as infrastructure_type,
       infra.name as infrastructure_name,
       collect(DISTINCT ss.name) as database_statefulsets,
       collect(DISTINCT cm.name) as database_configmaps,
       collect(DISTINCT infraSS.name) as infrastructure_statefulsets,
       collect(DISTINCT infraCM.name) as infrastructure_configmaps
```

**Returns:**
- `database`: Database name
- `single_instance`: Boolean indicating single instance deployment
- `infrastructure_type`: Type of infrastructure resource
- `infrastructure_name`: Name of infrastructure resource
- `database_statefulsets`: StatefulSets managed by the database
- `database_configmaps`: ConfigMaps used by the database
- `infrastructure_statefulsets`: StatefulSets owned by the infrastructure
- `infrastructure_configmaps`: ConfigMaps used by the infrastructure

## Best Practices

### 1. Query Performance
- Use indexes on frequently queried properties
- Limit result sets with WHERE clauses
- Use OPTIONAL MATCH for optional relationships
- Use collect() for aggregating related data

### 2. Relationship Analysis
- Always check for missing relationships
- Validate ownership consistency
- Monitor relationship creation errors
- Use fallback logic for missing identifiers

### 3. Data Quality
- Regular validation of ownership relationships
- Monitor for orphaned resources
- Track relationship creation failures
- Validate resource consistency

## Conclusion

The dynamic ownership relationships provide a flexible and accurate representation of Neo4j resource hierarchies. These queries enable comprehensive analysis of resource ownership, performance monitoring, and troubleshooting of Neo4j deployments in Kubernetes clusters. 
