#!/bin/bash
set -e

echo "Installing Kubernetes infrastructure components"
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

# Install Ingress Controller
echo "Installing NGINX Ingress Controller..."
if kubectl get namespace ingress-nginx > /dev/null 2>&1; then
  echo "Ingress Controller namespace already exists, applying updates..."
else
  echo "Creating Ingress Controller namespace..."
fi

kubectl apply -f k8s/infrastructure/ingress-nginx/deploy.yaml

echo ""
echo "Waiting for Ingress Controller to be ready..."
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=300s || true

echo ""
echo "Infrastructure installation status:"
echo ""
echo "Ingress Controller pods:"
kubectl get pods -n ingress-nginx
echo ""
echo "Ingress Controller service:"
kubectl get svc -n ingress-nginx ingress-nginx-controller
echo ""
echo "To get the external IP for DNS configuration:"
echo "  kubectl get svc -n ingress-nginx ingress-nginx-controller"

