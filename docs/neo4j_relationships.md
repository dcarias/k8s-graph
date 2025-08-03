# Neo4j Resource Relationships

This document outlines all the relationships between Neo4j resources that are automatically created by the kubegraph handlers.

## Overview

The Neo4j handlers create a rich graph of relationships between different Neo4j resources and Kubernetes resources, enabling comprehensive dependency tracking and resource management visualization.

## Relationship Types

### 1. Neo4jDatabase Relationships

**Neo4jDatabase → StatefulSet**
- **Relationship Type**: `MANAGED_BY`
- **Source**: Neo4jDatabase unified server statuses
- **Target**: StatefulSet by name
- **Description**: Links Neo4jDatabase to the StatefulSets that manage its pods

**Neo4jDatabase → ConfigMap**
- **Relationship Type**: `USES`
- **Source**: Neo4jDatabase applied settings
- **Target**: ConfigMap by name (Neo4jConfiguration)
- **Description**: Links Neo4jDatabase to the ConfigMap containing its Neo4j configuration

**Neo4jDatabase → Neo4jCluster**
- **Relationship Type**: `OWNS`
- **Source**: Neo4jDatabase spec (when SingleInstance is false)
- **Target**: Neo4jCluster by clusterId (targetHostClusterId or hostClusterId)
- **Description**: Links Neo4jDatabase to the Neo4jCluster it owns (for clustered databases)
- **Logic**: Created when `SingleInstance` field is `false`

**Neo4jDatabase → Neo4jSingleInstance**
- **Relationship Type**: `OWNS`
- **Source**: Neo4jDatabase spec (when SingleInstance is true)
- **Target**: Neo4jSingleInstance by dbid
- **Description**: Links Neo4jDatabase to the Neo4jSingleInstance it owns (for single instance databases)
- **Logic**: Created when `SingleInstance` field is `true`

### 2. Neo4jCluster Relationships

**Neo4jCluster → StatefulSet**
- **Relationship Type**: `OWNS`
- **Source**: Neo4jCluster unified server statuses
- **Target**: StatefulSet by name
- **Description**: Links Neo4jCluster to the StatefulSets it owns and manages

**Neo4jCluster → ConfigMap**
- **Relationship Type**: `USES`
- **Source**: Neo4jCluster applied settings
- **Target**: ConfigMap by name (Neo4jConfiguration)
- **Description**: Links Neo4jCluster to the ConfigMap containing its Neo4j configuration

**Neo4jCluster → PVC**
- **Relationship Type**: `OWNS`
- **Source**: PVC labels (clusterId)
- **Target**: PVC by uid
- **Description**: Links Neo4jCluster to the PVCs it owns (based on labels)

### 3. Neo4jSingleInstance Relationships

**Neo4jSingleInstance → StatefulSet**
- **Relationship Type**: `OWNS`
- **Source**: Neo4jSingleInstance name
- **Target**: StatefulSet by name
- **Description**: Links Neo4jSingleInstance to the StatefulSet it owns and manages

**Neo4jSingleInstance → ConfigMap**
- **Relationship Type**: `USES`
- **Source**: Neo4jSingleInstance applied settings
- **Target**: ConfigMap by name (Neo4jConfiguration)
- **Description**: Links Neo4jSingleInstance to the ConfigMap containing its Neo4j configuration

**Neo4jSingleInstance → PVC**
- **Relationship Type**: `OWNS`
- **Source**: PVC labels (dbid)
- **Target**: PVC by uid
- **Description**: Links Neo4jSingleInstance to the PVCs it owns (based on labels)

### 4. PVC Relationships

**PersistentVolumeClaim → StatefulSet**
- **Relationship Type**: `USED_BY`
- **Source**: PVC name pattern matching StatefulSet name
- **Target**: StatefulSet by name
- **Description**: Links PVC to the StatefulSet that uses it
- **Pattern**: PVC `data-p-<clusterid>-<podIndex>-<volumeIndex>` → StatefulSet `p-<clusterid>-<podIndex>`

**PersistentVolumeClaim → Neo4jCluster**
- **Relationship Type**: `OWNED_BY`
- **Source**: PVC owner references or labels (clusterId)
- **Target**: Neo4jCluster by uid or clusterId
- **Description**: Links PVC to the Neo4jCluster that owns it

**PersistentVolumeClaim → Neo4jSingleInstance**
- **Relationship Type**: `OWNED_BY`
- **Source**: PVC owner references or labels (dbid)
- **Target**: Neo4jSingleInstance by uid or dbid
- **Description**: Links PVC to the Neo4jSingleInstance that owns it

**PersistentVolumeClaim → PersistentVolume**
- **Relationship Type**: `BOUND_TO`
- **Source**: PVC spec
- **Target**: PersistentVolume by name
- **Description**: Links PVC to the PersistentVolume it's bound to

## Relationship Hierarchy

```
Neo4jDatabase
├── OWNS → Neo4jCluster (when SingleInstance is false)
├── OWNS → Neo4jSingleInstance (when SingleInstance is true)
├── MANAGED_BY → StatefulSet
└── USES → ConfigMap

Neo4jCluster
├── OWNS → Neo4jDatabase (bidirectional ownership)
├── OWNS → StatefulSet
└── USES → ConfigMap

Neo4jSingleInstance
├── OWNS → Neo4jDatabase (bidirectional ownership)
├── OWNS → StatefulSet
└── USES → ConfigMap

StatefulSet
├── MANAGES → Pod
└── USES → Service

PersistentVolumeClaim
├── USED_BY → StatefulSet (via naming pattern)
├── OWNED_BY → Neo4jCluster (via owner references or labels)
├── OWNED_BY → Neo4jSingleInstance (via owner references or labels)
└── BOUND_TO → PersistentVolume
```

## Resource Properties for Relationship Matching

### Neo4jDatabase
- **Primary Key**: `uid`
- **Relationship Keys**: 
  - `dbid` (for OWNS relationships to Neo4jSingleInstance)
  - `targetHostClusterId` (for OWNS relationships to Neo4jCluster)
  - `name` (for MANAGED_BY relationships)

### Neo4jCluster
- **Primary Key**: `uid`
- **Relationship Keys**: 
  - `clusterId` (for OWNS relationships from Neo4jDatabase)
  - `name` (for OWNS relationships)

### Neo4jSingleInstance
- **Primary Key**: `uid`
- **Relationship Keys**: 
  - `dbid` (for OWNS relationships from Neo4jDatabase)
  - `name` (for OWNS relationships)

### StatefulSet
- **Primary Key**: `name`
- **Used in**: MANAGED_BY, OWNS relationships
- **Creates**: MANAGES relationships to Pods

### ConfigMap
- **Primary Key**: `name`
- **Used in**: USES relationships

### Pod
- **Primary Key**: `uid`
- **Used in**: MANAGES relationships (from StatefulSet)
- **Creates**: USES relationships to PVCs, Secrets, ConfigMaps

## Implementation Details

### Relationship Creation Logic

1. **Neo4jDatabase Handler**:
   - Creates `MANAGED_BY` relationships to StatefulSets from `UnifiedServerStatuses`
   - Creates `USES` relationships to ConfigMaps from `AppliedSettings.Neo4jConfiguration`
   - Creates `OWNS` relationships to Neo4jCluster from `Spec.TargetHostClusterId` (when SingleInstance is false)
   - Creates `OWNS` relationships to Neo4jSingleInstance from `Spec.DbId` (when SingleInstance is true)
   - **Dynamic Ownership Logic**: The handler checks the `SingleInstance` field to determine ownership:
     - If `SingleInstance` is `true`: Creates `OWNS` relationship to `Neo4jSingleInstance`
     - If `SingleInstance` is `false`: Creates `OWNS` relationship to `Neo4jCluster` (with fallback to `hostClusterId` if `targetHostClusterId` is empty)

2. **Neo4jCluster Handler**:
   - Creates `OWNS` relationships to StatefulSets from `UnifiedServerStatuses`
   - Creates `USES` relationships to ConfigMaps from `AppliedSettings.Neo4jConfiguration`

3. **Neo4jSingleInstance Handler**:
   - Creates `OWNS` relationships to StatefulSets based on resource name
   - Creates `USES` relationships to ConfigMaps from `AppliedSettings.Neo4jConfiguration`

4. **PVC Handler**:
   - Creates `BOUND_TO` relationships to PersistentVolumes from `Spec.VolumeName`
   - Creates `OWNED_BY` relationships to Neo4jCluster based on PVC name pattern or labels
   - Creates `OWNED_BY` relationships to Neo4jSingleInstance based on PVC name pattern or labels

### Error Handling

All relationship creation includes comprehensive error handling:
- Validates source and target resource existence
- Provides detailed error messages for debugging
- Gracefully handles missing relationship data
- Continues processing even if individual relationships fail
- Implements fallback logic for ownership relationship creation

## Benefits

### 1. Dependency Tracking
- Clear visualization of which resources depend on others
- Easy identification of resource ownership
- Tracking of configuration dependencies

### 2. Resource Management
- Understanding of resource hierarchies
- Identification of shared resources
- Mapping of infrastructure components

### 3. Troubleshooting
- Quick identification of related resources
- Understanding of resource dependencies
- Impact analysis of resource changes

### 4. Operational Insights
- Resource utilization patterns
- Configuration sharing analysis
- Infrastructure topology mapping

## Future Enhancements

### Potential Additional Relationships

1. **BackupSchedule → Neo4jDatabase**
   - Could be inferred from labels or annotations
   - Would enable backup dependency tracking

2. **Neo4jRole → Neo4jDatabase**
   - Could be inferred from role applications
   - Would enable access control mapping

3. **DomainName → Neo4jDatabase**
   - Could be inferred from DNS configurations
   - Would enable network topology mapping

4. **Cross-Cluster Relationships**
   - Relationships between different Neo4jClusters
   - Multi-cluster dependency tracking

### Enhanced Relationship Types

1. **Temporal Relationships**
   - Backup schedules and timing
   - Resource lifecycle events

2. **Performance Relationships**
   - Resource utilization patterns
   - Performance dependencies

3. **Security Relationships**
   - Access control mappings
   - Security policy dependencies

## Sample Cypher Queries

### Find All Resources Owned or Generated by a Neo4jDatabase

This query shows all resources that are owned by, managed by, or use a specific Neo4jDatabase:

```cypher
// Find all resources related to a specific Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNED_BY]->(owner)
RETURN db.name as Database,
       collect(DISTINCT ss.name) as StatefulSets,
       collect(DISTINCT cm.name) as ConfigMaps,
       collect(DISTINCT owner.name) as Owners
```

### Find All Neo4jDatabases and Their Complete Resource Graph

This query shows the complete resource graph for all Neo4jDatabases:

```cypher
// Find all Neo4jDatabases with their complete resource relationships
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNS]->(cluster:Neo4jCluster)
OPTIONAL MATCH (db)-[:OWNS]->(si:Neo4jSingleInstance)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)-[:USES]->(svc:Service)
OPTIONAL MATCH (ss)<-[:USED_BY]-(pvc:PersistentVolumeClaim)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (cluster)-[:OWNS]->(clusterSS:StatefulSet)
OPTIONAL MATCH (cluster)-[:USES]->(clusterCM:ConfigMap)
OPTIONAL MATCH (si)-[:OWNS]->(siSS:StatefulSet)
OPTIONAL MATCH (si)-[:USES]->(siCM:ConfigMap)
RETURN db.name as Database,
       db.phase as Phase,
       db.singleInstance as SingleInstance,
       db.coreCount as CoreCount,
       db.primariesCount as PrimariesCount,
       CASE 
         WHEN cluster IS NOT NULL THEN 'Neo4jCluster'
         WHEN si IS NOT NULL THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       COALESCE(cluster.name, si.name, 'None') as OwnerName,
       collect(DISTINCT ss.name) as DatabaseStatefulSets,
       collect(DISTINCT cm.name) as DatabaseConfigMaps,
       collect(DISTINCT pod.name) as Pods,
       collect(DISTINCT svc.name) as Services,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pv.name) as PersistentVolumes,
       collect(DISTINCT clusterSS.name) as ClusterStatefulSets,
       collect(DISTINCT clusterCM.name) as ClusterConfigMaps,
       collect(DISTINCT siSS.name) as SingleInstanceStatefulSets,
       collect(DISTINCT siCM.name) as SingleInstanceConfigMaps
ORDER BY db.name
```

### Find All Neo4jSingleInstances and Their Resources

This query shows all Neo4jSingleInstances with their related resources:

```cypher
// Find all Neo4jSingleInstances with their complete resource relationships
MATCH (si:Neo4jSingleInstance)
OPTIONAL MATCH (si)-[:OWNS]->(ss:StatefulSet)
OPTIONAL MATCH (si)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db:Neo4jDatabase)-[:OWNED_BY]->(si)
RETURN si.name as SingleInstance,
       si.phase as Phase,
       collect(DISTINCT ss.name) as StatefulSets,
       collect(DISTINCT cm.name) as ConfigMaps,
       collect(DISTINCT db.name) as OwnedDatabases
ORDER BY si.name
```

### Find Resources by Ownership Type

This query categorizes resources by their ownership relationship:

```cypher
// Find all Neo4jDatabases grouped by their ownership type
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:OWNED_BY]->(cluster:Neo4jCluster)
OPTIONAL MATCH (db)-[:OWNED_BY]->(si:Neo4jSingleInstance)
RETURN db.name as Database,
       CASE 
         WHEN cluster IS NOT NULL THEN 'Neo4jCluster'
         WHEN si IS NOT NULL THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       COALESCE(cluster.name, si.name, 'None') as OwnerName
ORDER BY OwnerType, Database
```

### Find Resource Dependencies for Troubleshooting

This query helps identify resource dependencies for troubleshooting:

```cypher
// Find resource dependencies for troubleshooting
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (cm)-[:CONTAINS]->(config:Config)
RETURN db.name as Database,
       db.phase as DatabasePhase,
       ss.name as StatefulSet,
       ss.readyReplicas as ReadyReplicas,
       ss.replicas as TotalReplicas,
       cm.name as ConfigMap,
       owner.name as Owner,
       owner.phase as OwnerPhase
```

### Find Neo4jSingleInstance Dependencies

This query helps identify Neo4jSingleInstance dependencies:

```cypher
// Find Neo4jSingleInstance dependencies for troubleshooting
MATCH (si:Neo4jSingleInstance {name: 'my-single-instance'})
OPTIONAL MATCH (si)-[:OWNS]->(ss:StatefulSet)
OPTIONAL MATCH (si)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db:Neo4jDatabase)-[:OWNED_BY]->(si)
RETURN si.name as SingleInstance,
       si.phase as SingleInstancePhase,
       ss.name as StatefulSet,
       ss.readyReplicas as ReadyReplicas,
       ss.replicas as TotalReplicas,
       cm.name as ConfigMap,
       collect(DISTINCT db.name) as OwnedDatabases
```

### Find Orphaned Resources

This query helps identify resources that might be orphaned:

```cypher
// Find Neo4jDatabases without proper ownership
MATCH (db:Neo4jDatabase)
WHERE NOT (db)-[:OWNED_BY]->()
RETURN db.name as OrphanedDatabase,
       db.phase as Phase,
       db.singleInstance as SingleInstance
```

### Find Resource Utilization Patterns

This query shows resource utilization patterns:

```cypher
// Find resource utilization patterns
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
RETURN db.name as Database,
       db.coreMemorySize as Memory,
       db.coreDiskSize as Disk,
       db.coreCount as Cores,
       count(DISTINCT ss) as StatefulSetCount,
       count(DISTINCT cm) as ConfigMapCount
ORDER BY db.coreMemorySize DESC
```

### Find All Resources Used by a Neo4jDatabase

This comprehensive query shows all Kubernetes resources used by a specific Neo4jDatabase:

```cypher
// Find all resources used by a specific Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)-[:USES]->(svc:Service)
OPTIONAL MATCH (owner)-[:OWNS]->(clusterPvc:PVC)
OPTIONAL MATCH (svc)-[:SELECTS]->(pod)
OPTIONAL MATCH (pvc:PVC)-[:BOUND_TO]->(pv:PV)
OPTIONAL MATCH (clusterPvc)-[:BOUND_TO]->(clusterPv:PV)
OPTIONAL MATCH (pv)-[:BOUND_TO]->(pvc)
OPTIONAL MATCH (clusterPv)-[:BOUND_TO]->(clusterPvc)
OPTIONAL MATCH (secret:Secret)
OPTIONAL MATCH (sc:StorageClass)
WHERE pv.storageClass = sc.name OR clusterPv.storageClass = sc.name
RETURN db.name as Database,
       db.phase as DatabasePhase,
       db.singleInstance as SingleInstance,
       collect(DISTINCT ss.name) as StatefulSets,
       collect(DISTINCT cm.name) as ConfigMaps,
       collect(DISTINCT owner.name) as Owners,
       collect(DISTINCT pod.name) as Pods,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT clusterPvc.name) as ClusterPVCs,
       collect(DISTINCT svc.name) as Services,
       collect(DISTINCT secret.name) as Secrets,
       collect(DISTINCT pv.name) as PersistentVolumes,
       collect(DISTINCT clusterPv.name) as ClusterPersistentVolumes,
       collect(DISTINCT sc.name) as StorageClasses
```

### Find All Disk, CPU, and Memory Resources for a Neo4jDatabase

This query shows comprehensive resource usage including disk, CPU, and memory for a specific Neo4jDatabase:

```cypher
// Find all disk, CPU, and memory resources for a Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:OWNS]->(cluster:Neo4jCluster)
OPTIONAL MATCH (db)-[:OWNS]->(si:Neo4jSingleInstance)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)<-[:USED_BY]-(pvc:PersistentVolumeClaim)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (pod)-[:SCHEDULED_ON]->(node:Node)
OPTIONAL MATCH (cluster)-[:OWNS]->(clusterSS:StatefulSet)
OPTIONAL MATCH (clusterSS)-[:MANAGES]->(clusterPod:Pod)
OPTIONAL MATCH (clusterSS)<-[:USED_BY]-(clusterPVC:PVC)
OPTIONAL MATCH (clusterPVC)-[:BOUND_TO]->(clusterPV:PersistentVolume)
OPTIONAL MATCH (clusterPod)-[:SCHEDULED_ON]->(clusterNode:Node)
OPTIONAL MATCH (si)-[:OWNS]->(siSS:StatefulSet)
OPTIONAL MATCH (siSS)-[:MANAGES]->(siPod:Pod)
OPTIONAL MATCH (siSS)<-[:USED_BY]-(siPVC:PVC)
OPTIONAL MATCH (siPVC)-[:BOUND_TO]->(siPV:PersistentVolume)
OPTIONAL MATCH (siPod)-[:SCHEDULED_ON]->(siNode:Node)
RETURN db.name as Database,
       db.phase as Phase,
       db.singleInstance as SingleInstance,
       
       // Database-level resources
       db.coreCount as DatabaseCoreCount,
       db.primariesCount as DatabasePrimariesCount,
       db.coreMemorySize as DatabaseMemorySize,
       db.coreDiskSize as DatabaseDiskSize,
       db.coreDiskVirtSize as DatabaseVirtualDiskSize,
       db.diskUsageBytes as DatabaseDiskUsageBytes,
       db.currentNodes as DatabaseNodeCount,
       db.currentRelationships as DatabaseRelationshipCount,
       
       // Infrastructure owner
       CASE 
         WHEN cluster IS NOT NULL THEN 'Neo4jCluster'
         WHEN si IS NOT NULL THEN 'Neo4jSingleInstance'
         ELSE 'None'
       END as InfrastructureType,
       COALESCE(cluster.name, si.name, 'None') as InfrastructureName,
       
       // Infrastructure-level resources
       COALESCE(cluster.coreCount, si.coreCount, 0) as InfrastructureCoreCount,
       COALESCE(cluster.coreMemorySize, si.coreMemorySize, 'None') as InfrastructureMemorySize,
       COALESCE(cluster.coreDiskSize, si.coreDiskSize, 0) as InfrastructureDiskSize,
       COALESCE(cluster.coreDiskVirtSize, si.coreDiskVirtSize, 0) as InfrastructureVirtualDiskSize,
       
       // Pod resources (database level)
       collect(DISTINCT {
         name: pod.name,
         node: node.name,
         cpuRequests: pod.cpuRequests,
         cpuLimits: pod.cpuLimits,
         memoryRequests: pod.memoryRequests,
         memoryLimits: pod.memoryLimits
       }) as DatabasePods,
       
       // Pod resources (infrastructure level)
       collect(DISTINCT {
         name: clusterPod.name,
         node: clusterNode.name,
         cpuRequests: clusterPod.cpuRequests,
         cpuLimits: clusterPod.cpuLimits,
         memoryRequests: clusterPod.memoryRequests,
         memoryLimits: clusterPod.memoryLimits
       }) as ClusterPods,
       
       collect(DISTINCT {
         name: siPod.name,
         node: siNode.name,
         cpuRequests: siPod.cpuRequests,
         cpuLimits: siPod.cpuLimits,
         memoryRequests: siPod.memoryRequests,
         memoryLimits: siPod.memoryLimits
       }) as SingleInstancePods,
       
       // Storage resources (database level)
       collect(DISTINCT {
         name: pvc.name,
         size: pvc.size,
         storageClass: pvc.storageClass,
         volumeName: pv.name,
         volumeSize: pv.capacity
       }) as DatabaseStorage,
       
       // Storage resources (infrastructure level)
       collect(DISTINCT {
         name: clusterPVC.name,
         size: clusterPVC.size,
         storageClass: clusterPVC.storageClass,
         volumeName: clusterPV.name,
         volumeSize: clusterPV.capacity
       }) as ClusterStorage,
       
       collect(DISTINCT {
         name: siPVC.name,
         size: siPVC.size,
         storageClass: siPVC.storageClass,
         volumeName: siPV.name,
         volumeSize: siPV.capacity
       }) as SingleInstanceStorage,
       
       // Node resources
       collect(DISTINCT {
         name: node.name,
         cpu: node.cpu,
         memory: node.memory,
         instanceType: node.instanceType
       }) as DatabaseNodes,
       
       collect(DISTINCT {
         name: clusterNode.name,
         cpu: clusterNode.cpu,
         memory: clusterNode.memory,
         instanceType: clusterNode.instanceType
       }) as ClusterNodes,
       
       collect(DISTINCT {
         name: siNode.name,
         cpu: siNode.cpu,
         memory: siNode.memory,
         instanceType: siNode.instanceType
       }) as SingleInstanceNodes
```

### Find Neo4jDatabase Resource Usage Summary

This query provides a concise summary of all disk, CPU, and memory resources used by a Neo4jDatabase:

```cypher
// Find Neo4jDatabase resource usage summary
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:OWNS]->(cluster:Neo4jCluster)
OPTIONAL MATCH (db)-[:OWNS]->(si:Neo4jSingleInstance)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)<-[:USED_BY]-(pvc:PersistentVolumeClaim)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (pod)-[:SCHEDULED_ON]->(node:Node)
OPTIONAL MATCH (cluster)-[:OWNS]->(clusterSS:StatefulSet)
OPTIONAL MATCH (clusterSS)-[:MANAGES]->(clusterPod:Pod)
OPTIONAL MATCH (clusterSS)<-[:USED_BY]-(clusterPVC:PVC)
OPTIONAL MATCH (clusterPVC)-[:BOUND_TO]->(clusterPV:PersistentVolume)
OPTIONAL MATCH (clusterPod)-[:SCHEDULED_ON]->(clusterNode:Node)
OPTIONAL MATCH (si)-[:OWNS]->(siSS:StatefulSet)
OPTIONAL MATCH (siSS)-[:MANAGES]->(siPod:Pod)
OPTIONAL MATCH (siSS)<-[:USED_BY]-(siPVC:PVC)
OPTIONAL MATCH (siPVC)-[:BOUND_TO]->(siPV:PersistentVolume)
OPTIONAL MATCH (siPod)-[:SCHEDULED_ON]->(siNode:Node)
RETURN db.name as Database,
       db.phase as Phase,
       db.singleInstance as SingleInstance,
       
       // Database specification
       db.coreCount as DatabaseCores,
       db.primariesCount as DatabasePrimaries,
       db.coreMemorySize as DatabaseMemory,
       db.coreDiskSize as DatabaseDisk,
       db.coreDiskVirtSize as DatabaseVirtualDisk,
       
       // Infrastructure specification
       CASE 
         WHEN cluster IS NOT NULL THEN 'Neo4jCluster'
         WHEN si IS NOT NULL THEN 'Neo4jSingleInstance'
         ELSE 'None'
       END as InfrastructureType,
       COALESCE(cluster.name, si.name, 'None') as InfrastructureName,
       COALESCE(cluster.coreCount, si.coreCount, 0) as InfrastructureCores,
       COALESCE(cluster.coreMemorySize, si.coreMemorySize, 'None') as InfrastructureMemory,
       COALESCE(cluster.coreDiskSize, si.coreDiskSize, 0) as InfrastructureDisk,
       
       // Actual usage
       db.diskUsageBytes as ActualDiskUsage,
       db.currentNodes as DatabaseNodes,
       db.currentRelationships as DatabaseRelationships,
       
       // Pod count and resources
       count(DISTINCT pod) + count(DISTINCT clusterPod) + count(DISTINCT siPod) as TotalPods,
       count(DISTINCT node) + count(DISTINCT clusterNode) + count(DISTINCT siNode) as TotalNodes,
       
       // Storage summary
       count(DISTINCT pvc) + count(DISTINCT clusterPVC) + count(DISTINCT siPVC) as TotalPVCs,
       count(DISTINCT pv) + count(DISTINCT clusterPV) + count(DISTINCT siPV) as TotalPVs,
       
       // Resource totals (if available)
       sum(COALESCE(pod.cpuRequests, 0)) + sum(COALESCE(clusterPod.cpuRequests, 0)) + sum(COALESCE(siPod.cpuRequests, 0)) as TotalCPURequests,
       sum(COALESCE(pod.memoryRequests, 0)) + sum(COALESCE(clusterPod.memoryRequests, 0)) + sum(COALESCE(siPod.memoryRequests, 0)) as TotalMemoryRequests,
       sum(COALESCE(pvc.size, 0)) + sum(COALESCE(clusterPVC.size, 0)) + sum(COALESCE(siPVC.size, 0)) as TotalStorageSize
```

### Find Neo4jDatabase Total Resource Usage

This query shows the total aggregated resource usage by adding up all pod resources:

```cypher
// Find Neo4jDatabase total resource usage
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:OWNS]->(cluster:Neo4jCluster)
OPTIONAL MATCH (db)-[:OWNS]->(si:Neo4jSingleInstance)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)<-[:USED_BY]-(pvc:PersistentVolumeClaim)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (cluster)-[:OWNS]->(clusterSS:StatefulSet)
OPTIONAL MATCH (clusterSS)-[:MANAGES]->(clusterPod:Pod)
OPTIONAL MATCH (clusterSS)<-[:USED_BY]-(clusterPVC:PVC)
OPTIONAL MATCH (clusterPVC)-[:BOUND_TO]->(clusterPV:PersistentVolume)
OPTIONAL MATCH (si)-[:OWNS]->(siSS:StatefulSet)
OPTIONAL MATCH (siSS)-[:MANAGES]->(siPod:Pod)
OPTIONAL MATCH (siSS)<-[:USED_BY]-(siPVC:PVC)
OPTIONAL MATCH (siPVC)-[:BOUND_TO]->(siPV:PersistentVolume)
RETURN db.name as Database,
       db.phase as Phase,
       db.singleInstance as SingleInstance,
       
       // Database specification
       db.coreCount as DatabaseCores,
       db.primariesCount as DatabasePrimaries,
       db.coreMemorySize as DatabaseMemory,
       db.coreDiskSize as DatabaseDisk,
       
       // Infrastructure specification
       CASE 
         WHEN cluster IS NOT NULL THEN 'Neo4jCluster'
         WHEN si IS NOT NULL THEN 'Neo4jSingleInstance'
         ELSE 'None'
       END as InfrastructureType,
       COALESCE(cluster.name, si.name, 'None') as InfrastructureName,
       COALESCE(cluster.coreCount, si.coreCount, 0) as InfrastructureCores,
       COALESCE(cluster.coreMemorySize, si.coreMemorySize, 'None') as InfrastructureMemory,
       COALESCE(cluster.coreDiskSize, si.coreDiskSize, 0) as InfrastructureDisk,
       
       // Total pod count
       count(DISTINCT pod) + count(DISTINCT clusterPod) + count(DISTINCT siPod) as TotalPods,
       
       // Total CPU usage (sum of all pod CPU requests)
       sum(COALESCE(pod.cpuRequests, 0)) + sum(COALESCE(clusterPod.cpuRequests, 0)) + sum(COALESCE(siPod.cpuRequests, 0)) as TotalCPURequests,
       sum(COALESCE(pod.cpuLimits, 0)) + sum(COALESCE(clusterPod.cpuLimits, 0)) + sum(COALESCE(siPod.cpuLimits, 0)) as TotalCPULimits,
       
       // Total Memory usage (sum of all pod memory requests)
       sum(COALESCE(pod.memoryRequests, 0)) + sum(COALESCE(clusterPod.memoryRequests, 0)) + sum(COALESCE(siPod.memoryRequests, 0)) as TotalMemoryRequests,
       sum(COALESCE(pod.memoryLimits, 0)) + sum(COALESCE(clusterPod.memoryLimits, 0)) + sum(COALESCE(siPod.memoryLimits, 0)) as TotalMemoryLimits,
       
       // Total Storage usage (sum of all PVC sizes)
       sum(COALESCE(pvc.size, 0)) + sum(COALESCE(clusterPVC.size, 0)) + sum(COALESCE(siPVC.size, 0)) as TotalStorageSize,
       
       // Actual database usage
       db.diskUsageBytes as ActualDatabaseDiskUsage,
       db.currentNodes as DatabaseNodeCount,
       db.currentRelationships as DatabaseRelationshipCount,
       
       // Resource breakdown by component
       count(DISTINCT pod) as DatabasePods,
       count(DISTINCT clusterPod) as ClusterPods,
       count(DISTINCT siPod) as SingleInstancePods,
       
       // Storage breakdown by component
       count(DISTINCT pvc) as DatabasePVCs,
       count(DISTINCT clusterPVC) as ClusterPVCs,
       count(DISTINCT siPVC) as SingleInstancePVCs,
       
       // CPU breakdown by component
       sum(COALESCE(pod.cpuRequests, 0)) as DatabaseCPURequests,
       sum(COALESCE(clusterPod.cpuRequests, 0)) as ClusterCPURequests,
       sum(COALESCE(siPod.cpuRequests, 0)) as SingleInstanceCPURequests,
       
       // Memory breakdown by component
       sum(COALESCE(pod.memoryRequests, 0)) as DatabaseMemoryRequests,
       sum(COALESCE(clusterPod.memoryRequests, 0)) as ClusterMemoryRequests,
       sum(COALESCE(siPod.memoryRequests, 0)) as SingleInstanceMemoryRequests,
       
       // Storage breakdown by component
       sum(COALESCE(pvc.size, 0)) as DatabaseStorageSize,
       sum(COALESCE(clusterPVC.size, 0)) as ClusterStorageSize,
       sum(COALESCE(siPVC.size, 0)) as SingleInstanceStorageSize
```

### Find All Neo4jDatabases with Related Resources Only

This query shows only the resources that are actually related to Neo4jDatabases:

```cypher
// Find all Neo4jDatabases with only related resources
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)-[:USES]->(svc:Service)
OPTIONAL MATCH (ss)<-[:USED_BY]-(pvc:PersistentVolumeClaim)
OPTIONAL MATCH (svc)-[:SELECTS]->(pod)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (pv)-[:BOUND_TO]->(pvc)
RETURN db.name as Database,
       db.phase as Phase,
       db.singleInstance as SingleInstance,
       db.coreMemorySize as Memory,
       db.coreDiskSize as Disk,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'Unknown'
       END as OwnerType,
       owner.name as OwnerName,
       count(DISTINCT ss) as StatefulSetCount,
       count(DISTINCT pod) as PodCount,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT svc) as ServiceCount,
       count(DISTINCT cm) as ConfigMapCount,
       count(DISTINCT pv) as PVCount
ORDER BY db.name
```

### Find All Neo4jDatabases with Complete Resource Inventory

This query provides a complete inventory of all resources for all Neo4jDatabases:

```cypher
// Find all Neo4jDatabases with complete resource inventory
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNS]->(owner)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (ss)-[:USES]->(svc:Service)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (svc)-[:SELECTS]->(pod)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (pv)-[:BOUND_TO]->(pvc)
OPTIONAL MATCH (secret:Secret)
WITH db, ss, cm, owner, pod, svc, pvc, pv, secret
OPTIONAL MATCH (sc:StorageClass)
WHERE (pvc IS NOT NULL AND pvc.storageClass = sc.name) OR (pv IS NOT NULL AND pv.storageClass = sc.name)
RETURN db.name as Database,
       db.phase as Phase,
       db.singleInstance as SingleInstance,
       db.targetHostClusterId as TargetHostClusterId,
       db.dbid as DatabaseId,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'Unknown'
       END as OwnerType,
       owner.name as OwnerName,
       count(DISTINCT ss) as StatefulSetCount,
       count(DISTINCT pod) as PodCount,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT svc) as ServiceCount,
       count(DISTINCT cm) as ConfigMapCount,
       count(DISTINCT secret) as SecretCount,
       count(DISTINCT pv) as PVCount,
       count(DISTINCT sc) as StorageClassCount
ORDER BY db.name
```

### Debug Neo4jDatabase Properties

This query helps debug what properties are actually available on Neo4jDatabase nodes:

```cypher
// Debug Neo4jDatabase properties
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:OWNS]->(owner)
RETURN db.name as Database,
       db.phase as Phase,
       db.targetHostClusterId as TargetHostClusterId,
       db.dbid as DatabaseId,
       db.hostClusterId as HostClusterId,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       owner.clusterId as OwnerClusterId,
       owner.dbid as OwnerDatabaseId
ORDER BY db.name
```

### Simple Test Query

This query provides a simple test to see what relationships are actually working:

```cypher
// Simple test query to see what's working
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNS]->(owner)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
RETURN db.name as Database,
       db.singleInstance as SingleInstance,
       db.targetHostClusterId as TargetHostClusterId,
       db.dbid as DatabaseId,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       count(DISTINCT ss) as StatefulSetCount,
       count(DISTINCT cm) as ConfigMapCount,
       count(DISTINCT pvc) as PVCCount
ORDER BY db.name
```

### Comprehensive Debug Query

This query helps debug all the issues step by step:

```cypher
// Comprehensive debug query
MATCH (db:Neo4jDatabase {name: 'e1229598'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNS]->(owner)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (pv)-[:BOUND_TO]->(pvc)
RETURN db.name as Database,
       db.singleInstance as SingleInstance,
       db.targetHostClusterId as TargetHostClusterId,
       db.dbid as DatabaseId,
       db.hostClusterId as HostClusterId,
       collect(DISTINCT ss.name) as StatefulSets,
       collect(DISTINCT cm.name) as ConfigMaps,
       collect(DISTINCT owner.name) as Owners,
       collect(DISTINCT owner.clusterId) as OwnerClusterIds,
       collect(DISTINCT owner.dbid) as OwnerDatabaseIds,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pv.name) as PVs,
       count(DISTINCT ss) as StatefulSetCount,
       count(DISTINCT cm) as ConfigMapCount,
       count(DISTINCT owner) as OwnerCount,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT pv) as PVCount
```

### Debug All Neo4jDatabases

This query shows all Neo4jDatabases and their key properties:

```cypher
// Debug all Neo4jDatabases
MATCH (db:Neo4jDatabase)
OPTIONAL MATCH (db)-[:OWNS]->(owner)
RETURN db.name as Database,
       db.singleInstance as SingleInstance,
       db.targetHostClusterId as TargetHostClusterId,
       db.dbid as DatabaseId,
       db.hostClusterId as HostClusterId,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       owner.clusterId as OwnerClusterId,
       owner.dbid as OwnerDatabaseId
ORDER BY db.name
```

### Debug All PVCs

This query shows all PVCs and their relationships:

```cypher
// Debug all PVCs
MATCH (pvc:PersistentVolumeClaim)
OPTIONAL MATCH (pvc)-[:USED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN pvc.name as PVCName,
       pvc.namespace as PVCNamespace,
       pvc.storageClass as StorageClass,
       pvc.status as PVCStatus,
       ss.name as StatefulSetUser,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       owner.clusterId as OwnerClusterId,
       owner.dbid as OwnerDatabaseId,
       pv.name as PVName,
       pv.phase as PVPhase
ORDER BY pvc.name
```

### Debug All Neo4jClusters and Neo4jSingleInstances

This query shows all Neo4jClusters and Neo4jSingleInstances:

```cypher
// Debug all Neo4jClusters and Neo4jSingleInstances
MATCH (cluster:Neo4jCluster)
RETURN cluster.name as Name,
       cluster.clusterId as ClusterId,
       cluster.dbid as DatabaseId,
       'Neo4jCluster' as Type
UNION ALL
MATCH (si:Neo4jSingleInstance)
RETURN si.name as Name,
       si.clusterId as ClusterId,
       si.dbid as DatabaseId,
       'Neo4jSingleInstance' as Type
ORDER BY Name
```

### Check if Neo4jCluster/Neo4jSingleInstance Resources Exist

This query checks if the expected Neo4jCluster/Neo4jSingleInstance resources exist:

```cypher
// Check if expected resources exist
MATCH (db:Neo4jDatabase {name: 'e1229598'})
WITH db, db.hostClusterId as expectedClusterId, db.dbid as expectedDatabaseId
OPTIONAL MATCH (cluster:Neo4jCluster {clusterId: expectedClusterId})
OPTIONAL MATCH (si:Neo4jSingleInstance {dbid: expectedDatabaseId})
RETURN db.name as Database,
       db.singleInstance as SingleInstance,
       db.hostClusterId as ExpectedClusterId,
       db.dbid as ExpectedDatabaseId,
       cluster.name as FoundClusterName,
       cluster.clusterId as FoundClusterId,
       si.name as FoundSingleInstanceName,
       si.dbid as FoundSingleInstanceId,
       CASE 
         WHEN cluster IS NOT NULL THEN 'Neo4jCluster Found'
         WHEN si IS NOT NULL THEN 'Neo4jSingleInstance Found'
         ELSE 'No Resource Found'
       END as Status
```

### Debug PVCs for Specific Database

This query checks what PVCs exist and their relationships for the e1229598 database:

```cypher
// Debug PVCs for e1229598 database
MATCH (db:Neo4jDatabase {name: 'e1229598'})
OPTIONAL MATCH (db)-[:OWNS]->(owner)
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN db.name as Database,
       db.hostClusterId as HostClusterId,
       owner.name as OwnerName,
       owner.clusterId as OwnerClusterId,
       collect(DISTINCT ss.name) as StatefulSets,
       collect(DISTINCT pvc.name) as PVCsUsedByStatefulSets,
       collect(DISTINCT pvc.name) as PVCsOwnedByOwner,
       collect(DISTINCT pv.name) as PVs,
       count(DISTINCT pvc) as TotalPVCCount
```

### Debug All PVCs with Naming Pattern

This query shows all PVCs that might be related to Neo4j resources based on naming patterns:

```cypher
// Debug all PVCs with naming pattern
MATCH (pvc:PersistentVolumeClaim)
WHERE pvc.name STARTS WITH 'data-p-'
WITH pvc, split(substring(pvc.name, 6), '-') as parts
WHERE size(parts) >= 3
OPTIONAL MATCH (pvc)-[:USED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN pvc.name as PVCName,
       pvc.namespace as PVCNamespace,
       parts[0] as ClusterOrSingleInstanceId,
       parts[1] as PodIndex,
       parts[2] as VolumeIndex,
       ss.name as StatefulSetUser,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       pv.name as PVName,
       pv.phase as PVPhase
ORDER BY pvc.name
```

### Debug StatefulSets for e1229598 Database

This query shows what StatefulSets exist for the e1229598 database:

```cypher
// Debug StatefulSets for e1229598 database
MATCH (db:Neo4jDatabase {name: 'e1229598'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN db.name as Database,
       collect(DISTINCT ss.name) as StatefulSets,
       collect(DISTINCT ss.readyReplicas) as ReadyReplicas,
       collect(DISTINCT ss.replicas) as TotalReplicas,
       collect(DISTINCT pod.name) as Pods,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pv.name) as PVs,
       count(DISTINCT ss) as StatefulSetCount,
       count(DISTINCT pod) as PodCount,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT pv) as PVCount
```

### Debug All StatefulSets

This query shows all StatefulSets and their relationships:

```cypher
// Debug all StatefulSets
MATCH (ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (db:Neo4jDatabase)-[:MANAGED_BY]->(ss)
RETURN ss.name as StatefulSetName,
       ss.namespace as StatefulSetNamespace,
       ss.readyReplicas as ReadyReplicas,
       ss.replicas as TotalReplicas,
       collect(DISTINCT pod.name) as Pods,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pv.name) as PVs,
       collect(DISTINCT db.name) as ManagedDatabases,
       count(DISTINCT pod) as PodCount,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT pv) as PVCount
ORDER BY ss.name
```

### Simple PVC Check

This query shows all PVCs and their basic properties:

```cypher
// Simple PVC check
MATCH (pvc:PersistentVolumeClaim)
RETURN pvc.name as PVCName,
       pvc.namespace as PVCNamespace,
       pvc.storageClass as StorageClass,
       pvc.status as PVCStatus,
       pvc.volumeName as VolumeName,
       pvc.labels as PVCLabels
ORDER BY pvc.name
```

### Check PVCs for e1229598 Pattern

This query checks for PVCs that might be related to the e1229598 database:

```cypher
// Check PVCs for e1229598 pattern
MATCH (pvc:PersistentVolumeClaim)
WHERE pvc.name CONTAINS 'e1229598' OR pvc.name CONTAINS 'e1229598-12f1'
OPTIONAL MATCH (pvc)-[:USED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN pvc.name as PVCName,
       pvc.namespace as PVCNamespace,
       pvc.storageClass as StorageClass,
       pvc.status as PVCStatus,
       ss.name as StatefulSetUser,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       pv.name as PVName,
       pv.phase as PVPhase,
       pvc.labels as PVCLabels
ORDER BY pvc.name
```

### Check StatefulSet Naming Pattern

This query checks what StatefulSets exist and compares with expected naming:

```cypher
// Check StatefulSet naming pattern
MATCH (ss:StatefulSet)
WHERE ss.name CONTAINS 'e1229598' OR ss.name CONTAINS 'e1229598-12f1'
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:USED_BY]->(ss)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN ss.name as StatefulSetName,
       ss.namespace as StatefulSetNamespace,
       ss.readyReplicas as ReadyReplicas,
       ss.replicas as TotalReplicas,
       collect(DISTINCT pod.name) as Pods,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pv.name) as PVs,
       count(DISTINCT pod) as PodCount,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT pv) as PVCount
ORDER BY ss.name
```

### Debug PVC to StatefulSet Relationship

This query specifically checks why the PVC doesn't have a USED_BY relationship:

```cypher
// Debug PVC to StatefulSet relationship
MATCH (pvc:PersistentVolumeClaim {name: 'data-p-e1229598-12f1-0001-0'})
WITH pvc, split(substring(pvc.name, 6), '-') as parts
WHERE size(parts) >= 4
WITH pvc, parts, 
     parts[0] as prefix,
     parts[1] as clusterId,
     parts[2] as podIndex,
     parts[3] as volumeIndex
WITH pvc, prefix, clusterId, podIndex, volumeIndex,
     prefix + '-' + clusterId + '-' + podIndex as expectedStatefulSetName
OPTIONAL MATCH (ss:StatefulSet {name: expectedStatefulSetName})
OPTIONAL MATCH (pvc)-[:USED_BY]->(actualSS:StatefulSet)
OPTIONAL MATCH (pvc)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
RETURN pvc.name as PVCName,
       expectedStatefulSetName as ExpectedStatefulSet,
       ss.name as FoundStatefulSet,
       actualSS.name as ActualStatefulSetUser,
       CASE 
         WHEN owner:Neo4jCluster THEN 'Neo4jCluster'
         WHEN owner:Neo4jSingleInstance THEN 'Neo4jSingleInstance'
         ELSE 'No Owner'
       END as OwnerType,
       owner.name as OwnerName,
       pv.name as PVName,
       pv.phase as PVPhase
```

### Debug StorageClasses

This query shows all StorageClasses and their usage:

```cypher
// Debug StorageClasses
MATCH (sc:StorageClass)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)
WHERE pvc.storageClass = sc.name
OPTIONAL MATCH (pv:PersistentVolume)
WHERE pv.storageClass = sc.name
RETURN sc.name as StorageClassName,
       sc.provisioner as Provisioner,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pv.name) as PVs,
       count(DISTINCT pvc) as PVCCount,
       count(DISTINCT pv) as PVCount
ORDER BY sc.name
```

### Debug StorageClasses for e1229598 Database

This query specifically checks StorageClasses used by the e1229598 database:

```cypher
// Debug StorageClasses for e1229598 database
MATCH (db:Neo4jDatabase {name: 'e1229598'})
OPTIONAL MATCH (db)-[:OWNS]->(owner)
OPTIONAL MATCH (pvc:PersistentVolumeClaim)-[:OWNED_BY]->(owner)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)
OPTIONAL MATCH (sc:StorageClass)
WHERE pvc.storageClass = sc.name OR pv.storageClass = sc.name
RETURN db.name as Database,
       collect(DISTINCT pvc.name) as PVCs,
       collect(DISTINCT pvc.storageClass) as PVCCStorageClasses,
       collect(DISTINCT pv.name) as PVs,
       collect(DISTINCT pv.storageClass) as PVStorageClasses,
       collect(DISTINCT sc.name) as StorageClasses,
       count(DISTINCT sc) as StorageClassCount
```

### Simple StorageClass Check

This query shows all StorageClasses in the database:

```cypher
// Simple StorageClass check
MATCH (sc:StorageClass)
RETURN sc.name as StorageClassName,
       sc.provisioner as Provisioner,
       sc.reclaimPolicy as ReclaimPolicy,
       sc.volumeBindingMode as VolumeBindingMode,
       sc.allowVolumeExpansion as AllowVolumeExpansion
ORDER BY sc.name
```
