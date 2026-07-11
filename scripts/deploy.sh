#!/bin/bash
set -e

echo "Deploying application to Kubernetes cluster"
echo ""

# Verify kubectl is available
if ! command -v kubectl &> /dev/null; then
  echo "Error: kubectl is not installed"
  exit 1
fi

# Verify cluster access
echo "Verifying cluster access..."
kubectl cluster-info > /dev/null 2>&1 || {
  echo "Error: Cannot access Kubernetes cluster"
  echo "Make sure kubectl is configured correctly"
  exit 1
}

echo "Cluster: $(kubectl config view --minify -o jsonpath='{.clusters[0].name}')"
echo ""

# Apply Kubernetes manifests
echo "Applying Kubernetes manifests..."
kubectl apply -k k8s/

# Force pods to re-pull the :latest images. `apply` alone does NOT restart pods
# when the image tag is unchanged (same ":latest" string => no spec diff => no
# rollout), so freshly pushed code would otherwise never go live.
echo ""
echo "Restarting deployments to pick up newly pushed images..."
kubectl rollout restart deployment/message-service deployment/real-time-ntfn-service -n chat-app

echo ""
echo "Waiting for rollouts to complete..."
kubectl rollout status deployment/message-service -n chat-app --timeout=300s
kubectl rollout status deployment/real-time-ntfn-service -n chat-app --timeout=300s

echo ""
echo "Deployment status:"
echo ""
echo "Pods:"
kubectl get pods -n chat-app
echo ""
echo "Services:"
kubectl get svc -n chat-app
echo ""
echo "Deployments:"
kubectl get deployments -n chat-app
