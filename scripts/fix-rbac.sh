#!/bin/bash

# Script to diagnose and fix RBAC issues for kubegraph
# Usage: ./scripts/fix-rbac.sh [namespace]

set -e

NAMESPACE=${1:-kubegraph}
RELEASE_NAME=${2:-kubegraph}

echo "🔍 Diagnosing RBAC issues for kubegraph in namespace: $NAMESPACE"

# Check if namespace exists
if ! kubectl get namespace $NAMESPACE >/dev/null 2>&1; then
    echo "❌ Namespace $NAMESPACE does not exist"
    echo "Creating namespace..."
    kubectl create namespace $NAMESPACE
fi

# Check if ClusterRole exists
echo "📋 Checking ClusterRole..."
if kubectl get clusterrole $RELEASE_NAME >/dev/null 2>&1; then
    echo "✅ ClusterRole $RELEASE_NAME exists"
else
    echo "❌ ClusterRole $RELEASE_NAME does not exist"
fi

# Check if ClusterRoleBinding exists
echo "📋 Checking ClusterRoleBinding..."
if kubectl get clusterrolebinding $RELEASE_NAME >/dev/null 2>&1; then
    echo "✅ ClusterRoleBinding $RELEASE_NAME exists"
    echo "📋 ClusterRoleBinding details:"
    kubectl get clusterrolebinding $RELEASE_NAME -o yaml | grep -A 10 subjects
else
    echo "❌ ClusterRoleBinding $RELEASE_NAME does not exist"
fi

# Check if ServiceAccount exists
echo "📋 Checking ServiceAccount..."
if kubectl get serviceaccount $RELEASE_NAME -n $NAMESPACE >/dev/null 2>&1; then
    echo "✅ ServiceAccount $RELEASE_NAME exists in namespace $NAMESPACE"
else
    echo "❌ ServiceAccount $RELEASE_NAME does not exist in namespace $NAMESPACE"
fi

# Test permissions
echo "🔐 Testing permissions..."
echo "Testing cluster-scoped resources:"

# Test cluster-scoped resources
for resource in nodes namespaces persistentvolumes limitranges; do
    if kubectl auth can-i list $resource --as=system:serviceaccount:$NAMESPACE:$RELEASE_NAME >/dev/null 2>&1; then
        echo "✅ Can list $resource"
    else
        echo "❌ Cannot list $resource"
    fi
done

echo "Testing namespace-scoped resources:"
# Test namespace-scoped resources
for resource in pods services deployments statefulsets daemonsets replicasets ingresses; do
    if kubectl auth can-i list $resource --as=system:serviceaccount:$NAMESPACE:$RELEASE_NAME >/dev/null 2>&1; then
        echo "✅ Can list $resource"
    else
        echo "❌ Cannot list $resource"
    fi
done

# Check kubegraph deployment
echo "📋 Checking kubegraph deployment..."
if kubectl get deployment $RELEASE_NAME -n $NAMESPACE >/dev/null 2>&1; then
    echo "✅ Deployment $RELEASE_NAME exists"
    echo "📋 ServiceAccount used by deployment:"
    kubectl get deployment $RELEASE_NAME -n $NAMESPACE -o jsonpath='{.spec.template.spec.serviceAccountName}'
    echo ""
else
    echo "❌ Deployment $RELEASE_NAME does not exist"
fi

# Check kubegraph logs for RBAC errors
echo "📋 Checking kubegraph logs for RBAC errors..."
if kubectl get deployment $RELEASE_NAME -n $NAMESPACE >/dev/null 2>&1; then
    echo "Recent logs (last 20 lines):"
    kubectl logs deployment/$RELEASE_NAME -n $NAMESPACE --tail=20 | grep -i "forbidden\|permission\|rbac" || echo "No RBAC-related errors found in recent logs"
else
    echo "Cannot check logs - deployment does not exist"
fi

echo ""
echo "🔧 To fix RBAC issues, try the following:"
echo "1. Reinstall the Helm chart with RBAC enabled:"
echo "   helm uninstall $RELEASE_NAME -n $NAMESPACE"
echo "   helm install $RELEASE_NAME ./helm/kubegraph -n $NAMESPACE --set rbac.create=true"
echo ""
echo "2. Or manually create RBAC resources:"
echo "   kubectl apply -f helm/kubegraph/templates/rbac.yaml"
echo ""
echo "3. Verify the fix:"
echo "   kubectl auth can-i list nodes --as=system:serviceaccount:$NAMESPACE:$RELEASE_NAME"
echo "   kubectl auth can-i list pods --as=system:serviceaccount:$NAMESPACE:$RELEASE_NAME" 
