# Troubleshooting kubegraph RBAC Issues

This guide helps you troubleshoot RBAC (Role-Based Access Control) permission issues when deploying kubegraph.

## Common RBAC Permission Errors

If you see errors like:
```
"Failed to watch" err="failed to list apps/v1, Resource=replicasets: replicasets.apps is forbidden: User \"system:serviceaccount:kubegraph:kubegraph\" cannot list resource \"replicasets\" in API group \"apps\" at the cluster scope"
```

This indicates that the service account doesn't have the necessary permissions to watch Kubernetes resources.

## Namespace Configuration

Before troubleshooting RBAC issues, ensure you're using the correct namespace:

### Option 1: Using Helm --namespace flag (Recommended)
```bash
# Install in a specific namespace
helm install kubegraph ./helm/kubegraph --namespace my-namespace --create-namespace

# Upgrade existing installation
helm upgrade kubegraph ./helm/kubegraph --namespace my-namespace
```

### Option 2: Using values.yaml configuration
```bash
# Install with namespace specified in values
helm install kubegraph ./helm/kubegraph --set namespace.name=my-namespace --set namespace.create=true
```

### Option 3: Using custom values file
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

## Debugging Steps

### 1. Check if RBAC Resources are Created

Verify that the ClusterRole and ClusterRoleBinding are created:

```bash
# Check if ClusterRole exists
kubectl get clusterrole kubegraph

# Check if ClusterRoleBinding exists
kubectl get clusterrolebinding kubegraph

# Check the ClusterRoleBinding details
kubectl describe clusterrolebinding kubegraph
```

### 2. Check Service Account

Verify the service account exists and is correctly named:

```bash
# Check if ServiceAccount exists (replace NAMESPACE with your namespace)
kubectl get serviceaccount kubegraph -n NAMESPACE

# Check ServiceAccount details
kubectl describe serviceaccount kubegraph -n NAMESPACE
```

### 3. Check RBAC Configuration

Verify the ClusterRoleBinding is correctly configured:

```bash
# Check ClusterRoleBinding subjects
kubectl get clusterrolebinding kubegraph -o yaml | grep -A 10 subjects
```

The subjects should show:
```yaml
subjects:
- kind: ServiceAccount
  name: kubegraph
  namespace: NAMESPACE  # Should match your namespace
```

### 4. Check ClusterRole Permissions

Verify the ClusterRole has the necessary permissions:

```bash
# Check ClusterRole rules
kubectl get clusterrole kubegraph -o yaml | grep -A 50 rules
```

### 5. Test Permissions

Test if the service account has the required permissions:

```bash
# Test listing pods (replace NAMESPACE with your namespace)
kubectl auth can-i list pods --as=system:serviceaccount:NAMESPACE:kubegraph

# Test listing deployments
kubectl auth can-i list deployments --as=system:serviceaccount:NAMESPACE:kubegraph

# Test listing replicasets
kubectl auth can-i list replicasets --as=system:serviceaccount:NAMESPACE:kubegraph
```

## Common Issues and Solutions

### Issue 1: RBAC Resources Not Created

**Symptoms**: No ClusterRole or ClusterRoleBinding found

**Solution**: Ensure RBAC is enabled in values:

```yaml
rbac:
  create: true
```

### Issue 2: Service Account Name Mismatch

**Symptoms**: Service account name in ClusterRoleBinding doesn't match actual service account

**Solution**: Check the service account name generation in `_helpers.tpl` and ensure it matches the deployment.

### Issue 3: Namespace Mismatch

**Symptoms**: ClusterRoleBinding references wrong namespace

**Solution**: Ensure the ClusterRoleBinding subjects reference the correct namespace where kubegraph is deployed.

### Issue 4: Missing API Group Permissions

**Symptoms**: Specific resources fail with "forbidden" errors

**Solution**: Check if the resource's API group is included in the ClusterRole rules.

## Manual RBAC Setup

If the Helm chart RBAC setup isn't working, you can manually create the RBAC resources:

```bash
# Create ClusterRole
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubegraph
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets", "namespaces", "serviceaccounts", "persistentvolumes", "persistentvolumeclaims", "endpoints", "events", "limitranges"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["storage.k8s.io"]
  resources: ["storageclasses"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["autoscaling"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["autoscaling/v2"]
  resources: ["horizontalpodautoscalers"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["autoscaling.k8s.io"]
  resources: ["verticalpodautoscalers"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["policy"]
  resources: ["poddisruptionbudgets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["networking.k8s.io"]
  resources: ["ingresses", "networkpolicies"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["neo4j.io"]
  resources: ["neo4jdatabases", "neo4jclusters", "neo4jsingleinstances", "neo4jroles", "backupschedules"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["ingressconfig.neo4j.io"]
  resources: ["ipaccesscontrols", "customendpoints", "domainnames"]
  verbs: ["get", "list", "watch"]
EOF

# Create ClusterRoleBinding (replace NAMESPACE with your namespace)
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubegraph
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubegraph
subjects:
- kind: ServiceAccount
  name: kubegraph
  namespace: NAMESPACE
EOF
```

## Verification Commands

After fixing RBAC issues, verify the setup:

```bash
# Check if kubegraph can access resources (replace NAMESPACE with your namespace)
kubectl auth can-i list pods --as=system:serviceaccount:NAMESPACE:kubegraph
kubectl auth can-i list deployments --as=system:serviceaccount:NAMESPACE:kubegraph
kubectl auth can-i list replicasets --as=system:serviceaccount:NAMESPACE:kubegraph

# Check kubegraph logs for permission errors (replace NAMESPACE with your namespace)
kubectl logs -f deployment/kubegraph -n NAMESPACE

# Check if informers are working
kubectl logs deployment/kubegraph -n NAMESPACE | grep "Setting up informer"
```

## Additional Resources

- [Kubernetes RBAC Documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Service Account Permissions](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
- [ClusterRole and ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#clusterrole-and-clusterrolebinding) 
