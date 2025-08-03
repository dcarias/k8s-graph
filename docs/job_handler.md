# Job Handler

The Job handler is responsible for monitoring and managing Job resources in Kubernetes clusters and storing their information in Neo4j.

## Overview

Jobs are Kubernetes resources that run a pod to completion. They are used for batch processing, one-time tasks, or any workload that needs to run to completion rather than running continuously.

## Properties Stored

The handler stores the following properties for each Job in Neo4j:

- `name`: The name of the Job
- `uid`: Unique identifier for the Job
- `namespace`: Kubernetes namespace where the Job is located
- `creationTimestamp`: When the Job was created
- `labels`: Kubernetes labels applied to the Job
- `annotations`: Kubernetes annotations applied to the Job
- `parallelism`: Number of pods to run in parallel
- `completions`: Number of successful completions required
- `backoffLimit`: Number of retries before marking as failed
- `activeDeadlineSeconds`: Maximum duration the job can run
- `ttlSecondsAfterFinished`: Time to live after completion
- `active`: Number of actively running pods
- `succeeded`: Number of successfully completed pods
- `failed`: Number of failed pods
- `startTime`: When the job started running
- `completionTime`: When the job completed
- `clusterName`: Name of the Kubernetes cluster
- `instanceHash`: Instance hash for multi-instance deployments

## Relationships Created

The handler creates the following relationships in Neo4j:

1. **OWNED_BY**: Links Job to its owner resources (e.g., CronJobs)
2. **MANAGES**: Links Job to the Pods it manages

### Ownership Relationships

The handler automatically creates `OWNED_BY` relationships based on the Job's `ownerReferences`. This typically includes:
- **CronJob**: Jobs created by CronJobs
- **Other controllers**: Any other resource that creates Jobs

### Pod Management Relationships

The handler creates `MANAGES` relationships with all Pods that match the Job's label selector. This provides a direct link between the Job and the Pods it's responsible for managing.

## Usage

The Job handler is automatically registered when the application starts. It will:

1. Watch for Job resources in all namespaces
2. Process create/update events to store Job information
3. Process delete events to remove Job information
4. Create relationships with related Kubernetes resources
5. Establish ownership relationships based on owner references

## Example Neo4j Queries

### Basic Job Queries

```cypher
// Find all Jobs
MATCH (job:Job) RETURN job

// Find Jobs in a specific namespace
MATCH (job:Job {namespace: "default"}) RETURN job

// Find Jobs with specific labels
MATCH (job:Job) WHERE job.labels.app = "batch-processor" RETURN job
```

### Relationship Queries

```cypher
// Find Jobs and their owners
MATCH (job:Job)-[:OWNED_BY]->(owner) RETURN job, owner

// Find Jobs and the pods they manage
MATCH (job:Job)-[:MANAGES]->(pod:Pod) RETURN job, pod

// Find CronJob -> Job -> Pod chain
MATCH (cronjob:CronJob)-[:CREATES]->(job:Job)-[:MANAGES]->(pod:Pod)
RETURN cronjob, job, pod
```

### Status Queries

```cypher
// Find running Jobs
MATCH (job:Job) WHERE job.active > 0 RETURN job.name, job.active

// Find completed Jobs
MATCH (job:Job) WHERE job.succeeded > 0 AND job.active = 0 RETURN job.name, job.succeeded

// Find failed Jobs
MATCH (job:Job) WHERE job.failed > 0 RETURN job.name, job.failed

// Find Jobs that exceeded their deadline
MATCH (job:Job) WHERE job.activeDeadlineSeconds IS NOT NULL 
AND job.startTime IS NOT NULL 
AND duration.between(datetime(job.startTime), datetime()).seconds > job.activeDeadlineSeconds
RETURN job.name, job.activeDeadlineSeconds
```

### Performance Queries

```cypher
// Find Jobs with high parallelism
MATCH (job:Job) WHERE job.parallelism > 5 RETURN job.name, job.parallelism

// Find Jobs that have been retried multiple times
MATCH (job:Job) WHERE job.failed > job.backoffLimit RETURN job.name, job.failed, job.backoffLimit
```

## Integration with Other Handlers

The Job handler works in conjunction with other handlers:

- **CronJob Handler**: Jobs are typically created by CronJobs
- **Pod Handler**: Jobs manage Pods
- **Node Handler**: Pods managed by Jobs run on Nodes

This creates a comprehensive graph of Kubernetes resources and their relationships. 
