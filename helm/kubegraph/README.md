# Kubegraph Helm Chart

This Helm chart deploys the Kubegraph application, which visualizes Kubernetes cluster resources in Neo4j.

## TODO

### Docker Repository and Build Workflow Setup

1. Create a Docker repository (e.g., on Docker Hub, GitHub Container Registry, or AWS ECR)
2. Set up GitHub Actions workflow for:
   - Building Docker image on push to main branch
   - Running tests
   - Pushing image to repository with appropriate tags
   - Updating Helm chart with new image version
3. Update `values.yaml` with the correct image repository
4. Add image pull secrets configuration if using private repository

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- Neo4j database (can be deployed separately or use an existing instance)

## Installing the Chart

To install the chart with the release name `kubegraph`:

```bash
helm install kubegraph ./helm/kubegraph
```

The command deploys kubegraph on the Kubernetes cluster with default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

### Namespace Configuration

You can specify the namespace in several ways:

1. **Using Helm `--namespace` flag** (recommended):
   ```bash
   helm install kubegraph ./helm/kubegraph --namespace my-namespace --create-namespace
   ```

2. **Using values.yaml configuration**:
   ```bash
   helm install kubegraph ./helm/kubegraph --set namespace.name=my-namespace --set namespace.create=true
   ```

3. **Using a custom values file**:
   ```yaml
   # my-values.yaml
   namespace:
     name: my-namespace
     create: true
     annotations:
       purpose: kubegraph-monitoring
     labels:
       environment: production
   ```
   ```bash
   helm install kubegraph ./helm/kubegraph -f my-values.yaml
   ```

## Uninstalling the Chart

To uninstall/delete the `kubegraph` deployment:

```bash
helm uninstall kubegraph
```

## Parameters

### Common parameters

| Name                | Description                                                                 | Value |
|---------------------|-----------------------------------------------------------------------------|-------|
| `nameOverride`      | String to partially override kubegraph.fullname template                    | `""`  |
| `fullnameOverride`  | String to fully override kubegraph.fullname template                        | `""`  |
| `replicaCount`      | Number of kubegraph replicas to deploy                                      | `1`   |

### Namespace parameters

| Name                    | Description                                                                 | Value   |
|-------------------------|-----------------------------------------------------------------------------|---------|
| `namespace.name`        | The namespace where kubegraph will be deployed                              | `""`    |
| `namespace.create`      | Whether to create the namespace if it doesn't exist                        | `false` |
| `namespace.annotations` | Annotations to add to the namespace                                         | `{}`    |
| `namespace.labels`      | Labels to add to the namespace                                              | `{}`    |

### Image parameters

| Name                | Description                                                                 | Value                |
|---------------------|-----------------------------------------------------------------------------|----------------------|
| `image.repository`  | kubegraph image repository                                                  | `dcarias/kubegraph` |
| `image.tag`         | kubegraph image tag                                                         | `latest`            |
| `image.pullPolicy`  | Image pull policy                                                           | `IfNotPresent`      |

### ServiceAccount parameters

| Name                     | Description                                                                 | Value   |
|--------------------------|-----------------------------------------------------------------------------|---------|
| `serviceAccount.create`  | Specifies whether a ServiceAccount should be created                        | `true`  |
| `serviceAccount.name`    | The name of the ServiceAccount to use                                       | `""`    |

### RBAC parameters

| Name           | Description                                                                 | Value   |
|----------------|-----------------------------------------------------------------------------|---------|
| `rbac.create`  | Specifies whether RBAC resources should be created                          | `true`  |

### Neo4j parameters

| Name                    | Description                                                                 | Value                |
|-------------------------|-----------------------------------------------------------------------------|----------------------|
| `neo4j.uri`            | Neo4j database URI                                                          | `neo4j://neo4j:7687`|
| `neo4j.username`       | Neo4j username                                                              | `neo4j`             |
| `neo4j.password`       | Neo4j password                                                              | `password`          |
| `neo4j.createSecret`   | Create a secret for Neo4j credentials                                       | `true`              |
| `neo4j.existingSecret` | Name of existing secret to use for Neo4j credentials                        | `""`                |

### Kubernetes parameters

| Name                          | Description                                                                 | Value     |
|-------------------------------|-----------------------------------------------------------------------------|-----------|
| `kubernetes.clusterName`      | Name of the Kubernetes cluster                                              | `default` |
| `kubernetes.useInClusterConfig` | Use in-cluster Kubernetes configuration                                  | `true`    |
| `kubernetes.configPath`       | Path to kubeconfig file (if not using in-cluster config)                    | `""`      |

### Resource parameters

| Name                | Description                                                                 | Value           |
|---------------------|-----------------------------------------------------------------------------|-----------------|
| `resources.limits.cpu`      | CPU resource limits                                                         | `500m`          |
| `resources.limits.memory`   | Memory resource limits                                                      | `512Mi`         |
| `resources.requests.cpu`    | CPU resource requests                                                       | `100m`          |
| `resources.requests.memory` | Memory resource requests                                                    | `128Mi`         |

## Configuration and installation details

### Using an existing Neo4j instance

To use an existing Neo4j instance, set the following values:

```bash
helm install kubegraph ./helm/kubegraph \
  --set neo4j.uri=neo4j://your-neo4j-host:7687 \
  --set neo4j.username=your-username \
  --set neo4j.password=your-password
```

### Using an existing Kubernetes configuration

To use an existing kubeconfig file:

```bash
helm install kubegraph ./helm/kubegraph \
  --set kubernetes.useInClusterConfig=false \
  --set kubernetes.configPath=/path/to/kubeconfig
```

### Using a custom cluster name

To set a custom cluster name:

```bash
helm install kubegraph ./helm/kubegraph \
  --set kubernetes.clusterName=my-cluster
```

### Creating a custom namespace with annotations

To create a custom namespace with specific annotations and labels:

```bash
helm install kubegraph ./helm/kubegraph \
  --set namespace.name=monitoring \
  --set namespace.create=true \
  --set namespace.annotations.purpose=kubegraph-monitoring \
  --set namespace.labels.environment=production
```

## Useful Cypher Queries

### List Pods and Their Node Instance Types

To list all pods for a specific deployment and the instance types of the nodes they're running on, including the cluster name:

```cypher
MATCH (p:Pod)-[:SCHEDULED_ON]->(n:Node)
WHERE p.name CONTAINS 'db-ingress'
RETURN p.name as pod_name, 
       n.name as node_name,
       p.clusterName as cluster_name,
       head(split(head(tail(split(n.labels, '"node.kubernetes.io/instance-type":"'))), '",')) as instance_type
```

This query will return:
- `pod_name`: The name of the pod
- `node_name`: The name of the node the pod is running on
- `cluster_name`: The name of the Kubernetes cluster
- `instance_type`: The AWS instance type (e.g., m5.xlarge, m5.large)

Replace 'db-ingress' with your deployment name to query different deployments. You can also filter by cluster name by adding `AND p.clusterName = 'your-cluster-name'` to the WHERE clause.
