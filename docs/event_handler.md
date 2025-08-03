# Event Handler

## Overview

The Event Handler tracks Kubernetes Events in Neo4j with automatic TTL-based cleanup. Events provide a chronological record of what has happened to objects in the cluster, including creation, updates, deletions, and various lifecycle events.

## Resource Information

- **API Group**: `core` (v1)
- **Version**: `v1`
- **Resource**: `events`
- **Namespaced**: Yes
- **Neo4j Label**: `Event`
- **TTL**: Configurable (default: 7 days)

## Properties Stored

### Core Properties
- `name`: The name of the Event
- `uid`: Unique identifier for the Event
- `namespace`: The namespace where the Event occurred
- `reason`: The reason for the event (e.g., "Scheduled", "Pulled", "Created", "Started")
- `message`: Detailed message describing the event
- `type`: Event type (Normal, Warning)
- `count`: Number of times this event has occurred
- `firstTimestamp`: When the event first occurred
- `lastTimestamp`: When the event last occurred
- `eventTime`: Precise event timestamp
- `source`: Source component that generated the event
- `involvedObject`: The object this event relates to (stored as string)
- `labels`: Kubernetes labels
- `annotations`: Kubernetes annotations
- `createdAt`: When this event was stored in Neo4j
- `clusterName`: The cluster where this event occurred
- `instanceHash`: Unique identifier for this application instance

## Relationships

### INVOLVES
- **From**: Event
- **To**: Any Kubernetes object (Pod, Service, Deployment, etc.)
- **Description**: Links the Event to the object it relates to

## Example Cypher Queries

### Basic Event Queries

#### Find all Events
```cypher
MATCH (e:Event) 
RETURN e.name, e.reason, e.type, e.namespace 
LIMIT 10
```

#### Find Events by cluster
```cypher
MATCH (e:Event {clusterName: 'duvaleks6-orch-0001'})
RETURN e.name, e.reason, e.type, e.namespace
LIMIT 10
```

#### Find Events by namespace
```cypher
MATCH (e:Event {namespace: 'default'})
RETURN e.name, e.reason, e.type, e.count
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

#### Find Events by reason
```cypher
MATCH (e:Event)
WHERE e.reason = 'Scheduled'
RETURN e.name, e.namespace, e.message, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 10
```

#### Find Warning Events
```cypher
MATCH (e:Event {type: 'Warning'})
RETURN e.name, e.reason, e.message, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

### Event Relationships

#### Find Events with their involved objects
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
RETURN e.name, e.reason, labels(obj)[0] as objectType, obj.name, e.namespace
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

#### Find Events by object type
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
WHERE labels(obj)[0] = 'Pod'
RETURN e.name, e.reason, obj.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 10
```

#### Find Events for specific object
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj {name: 'my-deployment'})
RETURN e.name, e.reason, e.message, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Count Events by object type
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
RETURN labels(obj)[0] as objectType, count(*) as eventCount
ORDER BY eventCount DESC
```

### Event Analysis

#### Find most active objects (by event count)
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
RETURN labels(obj)[0] as objectType, obj.name, count(*) as eventCount
ORDER BY eventCount DESC
LIMIT 20
```

#### Find recent Events (last 24 hours)
```cypher
MATCH (e:Event)
WHERE e.lastTimestamp > datetime() - duration({days: 1})
RETURN e.name, e.reason, e.type, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Find Events by time range
```cypher
MATCH (e:Event)
WHERE e.lastTimestamp > datetime('2025-06-28T00:00:00Z') 
  AND e.lastTimestamp < datetime('2025-06-29T00:00:00Z')
RETURN e.name, e.reason, e.type, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Find Events with high frequency (count > 1)
```cypher
MATCH (e:Event)
WHERE e.count > 1
RETURN e.name, e.reason, e.count, e.namespace, e.lastTimestamp
ORDER BY e.count DESC
LIMIT 20
```

### Pod Lifecycle Events

#### Find Pod creation events
```cypher
MATCH (e:Event)-[:INVOLVES]->(p:Pod)
WHERE e.reason IN ['Scheduled', 'Pulled', 'Created', 'Started']
RETURN e.name, e.reason, p.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

#### Find Pod failure events
```cypher
MATCH (e:Event)-[:INVOLVES]->(p:Pod)
WHERE e.reason IN ['Failed', 'BackOff', 'CrashLoopBackOff']
RETURN e.name, e.reason, e.message, p.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Find Pod lifecycle sequence for specific pod
```cypher
MATCH (e:Event)-[:INVOLVES]->(p:Pod {name: 'my-pod-123'})
RETURN e.reason, e.message, e.lastTimestamp
ORDER BY e.lastTimestamp ASC
```

### Deployment Events

#### Find Deployment scaling events
```cypher
MATCH (e:Event)-[:INVOLVES]->(d:Deployment)
WHERE e.reason IN ['ScalingReplicaSet', 'ScaledUp', 'ScaledDown']
RETURN e.name, e.reason, e.message, d.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Find Deployment rollout events
```cypher
MATCH (e:Event)-[:INVOLVES]->(d:Deployment)
WHERE e.reason IN ['RollingUpdate', 'RolloutComplete', 'RolloutFailed']
RETURN e.name, e.reason, e.message, d.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

### Service Events

#### Find Service events
```cypher
MATCH (e:Event)-[:INVOLVES]->(s:Service)
RETURN e.name, e.reason, e.message, s.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

### Custom Resource Events

#### Find IPAccessControl events
```cypher
MATCH (e:Event)-[:INVOLVES]->(iac:IPAccessControl)
RETURN e.name, e.reason, e.message, iac.name, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Find CustomEndpoint events
```cypher
MATCH (e:Event)-[:INVOLVES]->(ce:CustomEndpoint)
RETURN e.name, e.reason, e.message, ce.name, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

### Event Patterns and Anomalies

#### Find Events without relationships (orphaned events)
```cypher
MATCH (e:Event)
WHERE NOT EXISTS((e)-[:INVOLVES]->())
RETURN e.name, e.reason, e.involvedObject, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

#### Find Events by source component
```cypher
MATCH (e:Event)
WHERE e.source.component IS NOT NULL
RETURN e.source.component as component, count(*) as eventCount
ORDER BY eventCount DESC
```

#### Find Events with specific message patterns
```cypher
MATCH (e:Event)
WHERE e.message CONTAINS 'error' OR e.message CONTAINS 'failed'
RETURN e.name, e.reason, e.message, e.namespace, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

### Event Statistics

#### Event count by namespace
```cypher
MATCH (e:Event)
RETURN e.namespace, count(*) as eventCount
ORDER BY eventCount DESC
```

#### Event count by type
```cypher
MATCH (e:Event)
RETURN e.type, count(*) as eventCount
ORDER BY eventCount DESC
```

#### Event count by reason
```cypher
MATCH (e:Event)
RETURN e.reason, count(*) as eventCount
ORDER BY eventCount DESC
LIMIT 20
```

#### Events per hour (last 24 hours)
```cypher
MATCH (e:Event)
WHERE e.lastTimestamp > datetime() - duration({days: 1})
WITH e, datetime(e.lastTimestamp).hour as hour
RETURN hour, count(*) as eventCount
ORDER BY hour
```

### Event Cleanup and TTL

#### Find Events older than 7 days
```cypher
MATCH (e:Event)
WHERE e.createdAt < datetime() - duration({days: 7})
RETURN count(*) as oldEvents
```

#### Find Events that will be cleaned up soon
```cypher
MATCH (e:Event)
WHERE e.createdAt < datetime() - duration({days: 6})
RETURN e.name, e.reason, e.createdAt, e.lastTimestamp
ORDER BY e.createdAt ASC
LIMIT 20
```

### Complex Event Analysis

#### Find objects with the most recent events
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
WITH obj, max(e.lastTimestamp) as latestEvent
RETURN labels(obj)[0] as objectType, obj.name, latestEvent
ORDER BY latestEvent DESC
LIMIT 20
```

#### Find Events for objects in specific namespace
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
WHERE e.namespace = 'kube-system'
RETURN labels(obj)[0] as objectType, obj.name, e.reason, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
LIMIT 20
```

#### Find Events related to specific labels
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
WHERE obj.labels.app = 'myapp'
RETURN e.name, e.reason, labels(obj)[0] as objectType, obj.name, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

#### Find Events for objects with specific annotations
```cypher
MATCH (e:Event)-[:INVOLVES]->(obj)
WHERE obj.annotations.kubernetes\\.io/change-cause IS NOT NULL
RETURN e.name, e.reason, obj.annotations.kubernetes\\.io/change-cause as changeCause, e.lastTimestamp
ORDER BY e.lastTimestamp DESC
```

## Related Handlers

- **All Resource Handlers**: Events can reference any Kubernetes object
- **Pod Handler**: Most common event source
- **Deployment Handler**: Scaling and rollout events
- **Service Handler**: Service-related events
- **Custom Resource Handlers**: Events for custom resources

## Use Cases

- **Troubleshooting**: Understanding what happened to specific objects
- **Monitoring**: Tracking object lifecycle and health
- **Auditing**: Maintaining chronological records of cluster changes
- **Debugging**: Identifying patterns in failures or issues
- **Compliance**: Maintaining event logs for regulatory requirements
- **Performance Analysis**: Understanding object behavior over time

## Notes

- Events are automatically cleaned up based on the configured TTL (default: 7 days)
- Events are high-volume and can consume significant storage
- The `involvedObject` property contains the full object reference as a string
- Events are namespaced and only track objects in the same namespace
- The `instanceHash` helps identify which application instance processed the event
- Events without relationships indicate objects that don't have handlers or don't exist in Neo4j
- The TTL cleanup runs periodically in the background to remove expired events 
