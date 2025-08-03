# k8s-graph

A Kubernetes resource monitoring tool that synchronizes cluster resources into a Neo4j graph database for powerful relationship analysis and querying.

## Overview

k8s-graph monitors Kubernetes cluster resources and maintains a real-time graph representation in Neo4j. It focuses on **standard Kubernetes resources only** - providing comprehensive coverage of core workloads, services, storage, and cluster resources.

Perfect for:

- **Graph-based Analysis**: Query resource relationships using Cypher
- **Cluster Visualization**: Build custom dashboards showing resource connections
- **Dependency Mapping**: Understand how resources relate to each other
- **Historical Analysis**: Track resource changes over time
- **Troubleshooting**: Trace issues through resource relationships

## Features

- ✅ **Neo4j Integration**: Real-time synchronization to Neo4j graph database
- ✅ **Standard K8s Resources**: Pods, Deployments, Services, ConfigMaps, etc.
- ✅ **Relationship Mapping**: Automatically creates relationships between resources
- ✅ **Event Management**: Optional Kubernetes Events with TTL
- ✅ **Instance Management**: Handles multiple cluster instances safely
- ✅ **HTTP Status Server**: Built-in health and metrics endpoint
- ✅ **Cloud Native**: Designed for in-cluster and external deployment

## Quick Start

### Prerequisites
- Neo4j database (local or remote)
- Kubernetes cluster access
- Go 1.21+ (for building from source)

### Install

#### Using Go
```bash
go install github.com/dcarias/k8s-graph@latest
```

#### Using Helm
```bash
# Add the k8s-graph Helm repository
helm repo add k8s-graph https://dcarias.github.io/k8s-graph

# Update your local Helm chart repository cache
helm repo update

# Install k8s-graph
helm install k8s-graph k8s-graph/k8s-graph \
  --set neo4j.uri="neo4j://your-neo4j:7687" \
  --set neo4j.username="neo4j" \
  --set neo4j.password="your-password" \
  --set clusterName="production"

# Or install with custom values file
helm install k8s-graph k8s-graph/k8s-graph -f values.yaml
```

#### Sample Helm Values
```yaml
# values.yaml
clusterName: "production"
logLevel: "INFO"

neo4j:
  uri: "neo4j://neo4j-service:7687"
  username: "neo4j"
  password: "your-password"
  # Or use existing secret
  existingSecret: "neo4j-credentials"
  usernameKey: "username"
  passwordKey: "password"

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector: {}
tolerations: []
affinity: {}
```

### Basic Usage
```bash
# Connect to local Neo4j
k8s-graph --cluster-name=my-cluster

# Connect to remote Neo4j
k8s-graph \
  --neo4j-uri=neo4j://remote:7687 \
  --neo4j-username=your-user \
  --neo4j-password=your-password \
  --cluster-name=production
```

### Configuration

#### Command Line Options
```
Options:
  --cluster-name string        Name of the Kubernetes cluster (default "default")
  --event-ttl-days int         Number of days to retain events, 0 disables (default 7)
  --http-enabled               Enable HTTP server for status (default true)
  --http-port int              HTTP server port (default 8080)
  --kubeconfig string          Path to kubeconfig file (uses in-cluster config if empty)
  --log-level string           Log level: DEBUG, INFO, WARN, ERROR (default "INFO")
  --neo4j-password string      Neo4j password (default "password")
  --neo4j-uri string           Neo4j database URI (default "neo4j://localhost:7687")
  --neo4j-username string      Neo4j username (default "neo4j")
```

#### Environment Variables
```bash
export KUBECONFIG=/path/to/kubeconfig
export CLUSTER_NAME=production
export NEO4J_URI=neo4j://your-neo4j:7687
export NEO4J_USERNAME=neo4j
export NEO4J_PASSWORD=your-password
export LOG_LEVEL=INFO
export HTTP_ENABLED=true
export HTTP_PORT=8080
```

## Monitored Resources

k8s-graph monitors standard Kubernetes resources only:

### Core Workloads
- **Pods**: Lifecycle, relationships to controllers
- **Deployments**: Configuration, replica relationships
- **ReplicaSets**: Pod management relationships
- **DaemonSets**: Node deployment relationships
- **StatefulSets**: Ordered deployment relationships
- **Jobs**: Batch execution relationships
- **CronJobs**: Scheduled job relationships

### Services & Networking
- **Services**: Endpoint relationships, selectors
- **Endpoints**: Pod-to-service relationships
- **Ingress**: Service routing relationships
- **NetworkPolicies**: Security relationships

### Configuration & Storage
- **ConfigMaps**: Usage relationships with Pods
- **Secrets**: Usage relationships (metadata only)
- **PersistentVolumes**: Storage relationships
- **PersistentVolumeClaims**: Volume binding relationships
- **StorageClasses**: Storage configuration relationships

### RBAC & Policies
- **ServiceAccounts**: Pod authentication relationships
- **LimitRanges**: Resource constraint relationships

### Cluster Resources
- **Nodes**: Pod scheduling relationships
- **Namespaces**: Resource containment relationships

### Autoscaling
- **HorizontalPodAutoscalers**: Scaling relationships
- **VerticalPodAutoscalers**: Resource recommendation relationships
- **PodDisruptionBudgets**: Availability policy relationships

### Events (Optional)
- **Events**: Resource event relationships with TTL cleanup

## Neo4j Graph Structure

### Node Labels
Resources are stored with their Kubernetes kind as the label:
- `Pod`, `Deployment`, `Service`, `ConfigMap`, etc.

### Common Properties
All nodes include:
- `uid`: Kubernetes UID (unique identifier)
- `name`: Resource name
- `namespace`: Kubernetes namespace (if applicable)
- `clusterName`: Cluster identifier
- `createdAt`: Timestamp when added to graph
- `labels`: Kubernetes labels (as JSON)
- `annotations`: Kubernetes annotations (as JSON)

### Relationships
Automatic relationships are created:
- `OWNS`: Controller -> Controlled resources
- `USES`: Pod -> ConfigMap/Secret usage
- `SCHEDULES_ON`: Pod -> Node placement
- `SELECTS`: Service -> Pod relationships
- `INVOLVES`: Event -> Resource relationships

## Sample Cypher Queries

```cypher
// Find all pods in a deployment
MATCH (d:Deployment {name: "my-app"})-[:OWNS*]->(p:Pod)
RETURN d, p

// Find all services selecting a pod
MATCH (s:Service)-[:SELECTS]->(p:Pod {name: "my-pod"})
RETURN s, p

// Find pods using a specific ConfigMap
MATCH (p:Pod)-[:USES]->(cm:ConfigMap {name: "app-config"})
RETURN p, cm

// Find all resources in a namespace
MATCH (n) WHERE n.namespace = "production"
RETURN n

// Find recent events for a pod
MATCH (e:Event)-[:INVOLVES]->(p:Pod {name: "my-pod"})
WHERE e.createdAt > datetime() - duration('PT1H')
RETURN e, p
ORDER BY e.createdAt DESC
```

## Deployment

### Using Helm (Recommended)

The easiest way to deploy k8s-graph is using the official Helm chart:

```bash
helm repo add k8s-graph https://dcarias.github.io/k8s-graph
helm install k8s-graph k8s-graph/k8s-graph \
  --set neo4j.uri="neo4j://your-neo4j:7687" \
  --set neo4j.username="neo4j" \
  --set neo4j.password="your-password" \
  --set clusterName="production"
```

### Manual Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-graph
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: k8s-graph
  template:
    metadata:
      labels:
        app: k8s-graph
    spec:
      serviceAccountName: k8s-graph
      containers:
      - name: k8s-graph
        image: dcarias/k8s-graph:latest
        args:
          - --cluster-name=production
          - --log-level=INFO
        env:
        - name: NEO4J_URI
          value: "neo4j://neo4j-service:7687"
        - name: NEO4J_USERNAME
          valueFrom:
            secretKeyRef:
              name: neo4j-credentials
              key: username
        - name: NEO4J_PASSWORD
          valueFrom:
            secretKeyRef:
              name: neo4j-credentials
              key: password
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-graph
  namespace: monitoring
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-graph
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["batch"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["networking.k8s.io"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["policy"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["autoscaling"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k8s-graph
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: k8s-graph
subjects:
- kind: ServiceAccount
  name: k8s-graph
  namespace: monitoring
```

### Docker
```bash
docker run -d \
  --name k8s-graph \
  -v ~/.kube/config:/root/.kube/config:ro \
  -e NEO4J_URI=neo4j://your-neo4j:7687 \
  -e NEO4J_USERNAME=neo4j \
  -e NEO4J_PASSWORD=your-password \
  dcarias/k8s-graph:latest \
  --cluster-name=my-cluster
```

## HTTP Status Endpoints

When HTTP server is enabled (default), k8s-graph provides:

- **Health Check**: `GET /health` - Returns 200 if healthy
- **Metrics**: `GET /metrics` - Prometheus-compatible metrics
- **Status**: `GET /status` - Current configuration and Neo4j connection status

## Development

### Building
```bash
go build -o k8s-graph .
```

### Testing
```bash
go test ./...
```

### Running Locally
```bash
# Start local Neo4j (using Docker)
docker run --name neo4j -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/password \
  neo4j:latest

# Run k8s-graph
go run . --cluster-name=local --log-level=DEBUG
```

## Neo4j Browser

Access Neo4j Browser at `http://localhost:7474` to explore your cluster graph:

1. Connect with your Neo4j credentials
2. Run queries to explore resource relationships
3. Visualize cluster topology
4. Analyze resource dependencies

## Contributing

We welcome contributions! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Security

- **Secret Handling**: Secret data is never stored in Neo4j (metadata only)
- **RBAC**: Follows least-privilege principle (read-only access)
- **Neo4j Security**: Use secure connections and credentials for production

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/dcarias/k8s-graph/issues)
- **Discussions**: [GitHub Discussions](https://github.com/dcarias/k8s-graph/discussions)

## Focus

This project focuses on **standard Kubernetes resources only**, providing comprehensive monitoring of:

- Core workloads and their relationships
- Services and networking components
- Storage and configuration resources
- RBAC and security policies
- Cluster-level resources and autoscaling

This approach ensures the tool is broadly applicable to any Kubernetes cluster, regardless of specific distributions or add-ons.