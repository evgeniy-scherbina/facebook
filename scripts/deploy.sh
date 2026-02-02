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

echo ""
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/message-service -n chat-app || true
kubectl wait --for=condition=available --timeout=300s deployment/real-time-ntfn-service -n chat-app || true

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
