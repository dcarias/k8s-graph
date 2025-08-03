# CronJob Handler

The CronJob handler is responsible for monitoring and managing CronJob resources in Kubernetes clusters and storing their information in Neo4j.

## Overview

CronJobs are Kubernetes resources that create Jobs on a time-based schedule. They are used for recurring tasks, periodic maintenance, and automated batch processing.

## Properties Stored

The handler stores the following properties for each CronJob in Neo4j:

- `name`: The name of the CronJob
- `uid`: Unique identifier for the CronJob
- `namespace`: Kubernetes namespace where the CronJob is located
- `creationTimestamp`: When the CronJob was created
- `labels`: Kubernetes labels applied to the CronJob
- `annotations`: Kubernetes annotations applied to the CronJob
- `schedule`: Cron schedule expression (e.g., "0 0 * * *")
- `timeZone`: Timezone for the schedule
- `startingDeadlineSeconds`: Maximum time to wait for a job to start
- `concurrencyPolicy`: How to handle concurrent job executions
- `successfulJobsHistoryLimit`: Number of successful jobs to retain
- `failedJobsHistoryLimit`: Number of failed jobs to retain
- `suspend`: Whether the CronJob is suspended
- `lastScheduleTime`: When the last job was scheduled
- `lastSuccessfulTime`: When the last successful job completed
- `clusterName`: Name of the Kubernetes cluster
- `instanceHash`: Instance hash for multi-instance deployments

## Relationships Created

The handler creates the following relationships in Neo4j:

1. **OWNED_BY**: Links CronJob to its owner resources
2. **CREATES**: Links CronJob to the Jobs it creates

### Ownership Relationships

The handler automatically creates `OWNED_BY` relationships based on the CronJob's `ownerReferences`. This typically includes:
- **Other controllers**: Any resource that creates CronJobs

### Job Creation Relationships

The handler creates `CREATES` relationships with Jobs that are owned by the CronJob. This provides a direct link between the CronJob and the Jobs it creates.

## Usage

The CronJob handler is automatically registered when the application starts. It will:

1. Watch for CronJob resources in all namespaces
2. Process create/update events to store CronJob information
3. Process delete events to remove CronJob information
4. Create relationships with related Kubernetes resources
5. Establish ownership relationships based on owner references

## Example Neo4j Queries

### Basic CronJob Queries

```cypher
// Find all CronJobs
MATCH (cj:CronJob) RETURN cj

// Find CronJobs in a specific namespace
MATCH (cj:CronJob {namespace: "default"}) RETURN cj

// Find CronJobs with specific labels
MATCH (cj:CronJob) WHERE cj.labels.app = "backup" RETURN cj
```

### Relationship Queries

```cypher
// Find CronJobs and the jobs they create
MATCH (cj:CronJob)-[:CREATES]->(job:Job) RETURN cj, job

// Find CronJob -> Job -> Pod chain
MATCH (cj:CronJob)-[:CREATES]->(job:Job)-[:MANAGES]->(pod:Pod)
RETURN cj, job, pod

// Find CronJobs and their owners
MATCH (cj:CronJob)-[:OWNED_BY]->(owner) RETURN cj, owner
```

### Schedule Queries

```cypher
// Find CronJobs with specific schedules
MATCH (cj:CronJob) WHERE cj.schedule = "0 0 * * *" RETURN cj.name, cj.schedule

// Find CronJobs that run daily
MATCH (cj:CronJob) WHERE cj.schedule CONTAINS "0 0 * * *" RETURN cj.name, cj.schedule

// Find CronJobs that run hourly
MATCH (cj:CronJob) WHERE cj.schedule CONTAINS "0 * * * *" RETURN cj.name, cj.schedule
```

### Status Queries

```cypher
// Find suspended CronJobs
MATCH (cj:CronJob) WHERE cj.suspend = true RETURN cj.name, cj.namespace

// Find CronJobs that haven't run recently
MATCH (cj:CronJob) 
WHERE cj.lastScheduleTime IS NULL OR 
      datetime(cj.lastScheduleTime) < datetime() - duration({days: 7})
RETURN cj.name, cj.lastScheduleTime

// Find CronJobs with different concurrency policies
MATCH (cj:CronJob) 
RETURN cj.concurrencyPolicy, count(cj) as count
```

### Performance Queries

```cypher
// Find CronJobs with many retained jobs
MATCH (cj:CronJob) 
WHERE cj.successfulJobsHistoryLimit > 10 OR cj.failedJobsHistoryLimit > 10
RETURN cj.name, cj.successfulJobsHistoryLimit, cj.failedJobsHistoryLimit

// Find CronJobs with strict deadlines
MATCH (cj:CronJob) 
WHERE cj.startingDeadlineSeconds IS NOT NULL AND cj.startingDeadlineSeconds < 300
RETURN cj.name, cj.startingDeadlineSeconds
```

### Complex Workflow Queries

```cypher
// Find the complete workflow: CronJob -> Job -> Pod -> Node
MATCH (cj:CronJob)-[:CREATES]->(job:Job)-[:MANAGES]->(pod:Pod)-[:RUNS_ON]->(node:Node)
WHERE cj.schedule = "0 0 * * *"
RETURN cj.name, job.name, pod.name, node.name

// Find CronJobs with their recent job execution history
MATCH (cj:CronJob)-[:CREATES]->(job:Job)
WHERE job.creationTimestamp > datetime() - duration({days: 1})
RETURN cj.name, collect(job.name) as recentJobs, count(job) as jobCount
```

## Integration with Other Handlers

The CronJob handler works in conjunction with other handlers:

- **Job Handler**: CronJobs create Jobs
- **Pod Handler**: Jobs created by CronJobs manage Pods
- **Node Handler**: Pods managed by Jobs run on Nodes

This creates a comprehensive graph of Kubernetes resources and their relationships, allowing you to trace the complete execution chain from scheduled task to running pod. 
