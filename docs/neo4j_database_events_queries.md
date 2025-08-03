# Neo4jDatabase Events Queries

This document provides Cypher queries to retrieve all events related to a Neo4jDatabase, including direct events and events on related resources.

## Overview

Events in kubegraph are connected to resources via the `INVOLVES` relationship. Events can be filtered by type, reason, timestamp, and resource type.

## Basic Queries

### 1. Direct Events on Neo4jDatabase

```cypher
// Get all events directly related to a specific Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})<-[:INVOLVES]-(e:Event)
RETURN e.name as event_name,
       e.reason as reason,
       e.type as event_type,
       e.message as message,
       e.lastTimestamp as last_occurrence,
       e.count as occurrence_count
ORDER BY e.lastTimestamp DESC
```

### 2. All Events Related to Neo4jDatabase (Comprehensive)

```cypher
// Get all events related to a Neo4jDatabase and its resources
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (db)-[:USES]->(cm:ConfigMap)
OPTIONAL MATCH (db)-[:OWNS]->(owner)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)
OPTIONAL MATCH (pod)-[:SCHEDULED_ON]->(node:Node)
OPTIONAL MATCH (pod)-[:USES]->(pvc:PersistentVolumeClaim)
OPTIONAL MATCH (pod)-[:USES]->(secret:Secret)
OPTIONAL MATCH (pvc)-[:BOUND_TO]->(pv:PersistentVolume)

// Collect all related resources
WITH db, collect(ss) + collect(cm) + collect(owner) + collect(pod) + collect(node) + collect(pvc) + collect(secret) + collect(pv) as related_resources

// Get events for the database and all related resources
UNWIND [db] + related_resources as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL

RETURN e.name as event_name,
       e.reason as reason,
       e.type as event_type,
       e.message as message,
       e.lastTimestamp as last_occurrence,
       e.count as occurrence_count,
       labels(resource)[0] as resource_type,
       resource.name as resource_name
ORDER BY e.lastTimestamp DESC
```

## Filtered Queries

### 3. Recent Events (Last 24 Hours)

```cypher
// Get recent events for a Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL
  AND e.lastTimestamp > datetime() - duration({hours: 24})

RETURN e.name as event_name,
       e.reason as reason,
       e.type as event_type,
       e.message as message,
       e.lastTimestamp as last_occurrence,
       labels(resource)[0] as resource_type,
       resource.name as resource_name
ORDER BY e.lastTimestamp DESC
```

### 4. Warning and Error Events Only

```cypher
// Get warning and error events for a Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL
  AND e.type IN ['Warning', 'Error']

RETURN e.name as event_name,
       e.reason as reason,
       e.type as event_type,
       e.message as message,
       e.lastTimestamp as last_occurrence,
       labels(resource)[0] as resource_type,
       resource.name as resource_name
ORDER BY e.lastTimestamp DESC
```

### 5. Events by Resource Type

```cypher
// Get events grouped by resource type
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL

RETURN labels(resource)[0] as resource_type,
       resource.name as resource_name,
       count(e) as event_count,
       collect({
         name: e.name,
         reason: e.reason,
         type: e.type,
         message: e.message,
         lastTimestamp: e.lastTimestamp
       }) as events
ORDER BY resource_type, resource_name
```

## Advanced Queries

### 6. Event Timeline

```cypher
// Create an event timeline for a Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL

RETURN e.firstTimestamp as timestamp,
       e.reason as reason,
       e.type as event_type,
       e.message as message,
       labels(resource)[0] as resource_type,
       resource.name as resource_name
ORDER BY e.firstTimestamp ASC
```

### 7. Event Statistics

```cypher
// Get event statistics for a Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'my-database'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL

RETURN labels(resource)[0] as resource_type,
       resource.name as resource_name,
       count(e) as total_events,
       count(CASE WHEN e.type = 'Normal' THEN 1 END) as normal_events,
       count(CASE WHEN e.type = 'Warning' THEN 1 END) as warning_events,
       count(CASE WHEN e.type = 'Error' THEN 1 END) as error_events,
       collect(DISTINCT e.reason) as unique_reasons
ORDER BY total_events DESC
```

## Usage Examples

### Monitor Database Health

```cypher
// Monitor the health of a Neo4jDatabase through its events
MATCH (db:Neo4jDatabase {name: 'production-db'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL
  AND e.type IN ['Warning', 'Error']
  AND e.lastTimestamp > datetime() - duration({hours: 24})

RETURN e.reason as issue_type,
       e.message as description,
       e.lastTimestamp as occurred_at,
       labels(resource)[0] as affected_resource,
       resource.name as resource_name
ORDER BY e.lastTimestamp DESC
```

### Troubleshoot Database Issues

```cypher
// Troubleshoot issues with a Neo4jDatabase
MATCH (db:Neo4jDatabase {name: 'problematic-db'})
OPTIONAL MATCH (db)-[:MANAGED_BY]->(ss:StatefulSet)
OPTIONAL MATCH (ss)-[:MANAGES]->(pod:Pod)

UNWIND [db, ss, pod] as resource
MATCH (resource)<-[:INVOLVES]-(e:Event)
WHERE resource IS NOT NULL
  AND e.type = 'Error'

RETURN e.reason as error_type,
       e.message as error_details,
       e.lastTimestamp as error_time,
       e.count as occurrence_count,
       labels(resource)[0] as resource_type,
       resource.name as resource_name
ORDER BY e.lastTimestamp DESC
```

## Notes

- Events have a configurable TTL (default: 7 days)
- Use database name or database ID to filter events
- Events can be filtered by type, reason, and timestamp
- Related resources include StatefulSets, Pods, Nodes, PVCs, etc.
