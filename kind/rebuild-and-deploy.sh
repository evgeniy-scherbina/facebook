#!/bin/bash

# Script to rebuild Docker images and redeploy to Kubernetes
# Usage: ./rebuild-and-deploy.sh

set -e

echo "🔨 Building Docker images..."

# Build message service
echo "Building message-service..."
docker build -f services/message/Dockerfile -t message-service:latest .

# Build real-time notification service
echo "Building real-time-ntfn-service..."
docker build -f services/real-time-ntfn/Dockerfile -t real-time-ntfn-service:latest .

echo ""
echo "📦 Loading images into Kind cluster..."

# Get the Kind cluster name
CLUSTER_NAME=$(kind get clusters | head -n 1)
if [ -z "$CLUSTER_NAME" ]; then
    echo "❌ No Kind cluster found. Please create one first."
    exit 1
fi

echo "Using Kind cluster: $CLUSTER_NAME"

# Load images into Kind
kind load docker-image message-service:latest --name "$CLUSTER_NAME"
kind load docker-image real-time-ntfn-service:latest --name "$CLUSTER_NAME"

echo ""
echo "🚀 Restarting deployments to use new images..."

# Restart deployments to pull new images
kubectl rollout restart deployment/message-service -n chat-app
kubectl rollout restart deployment/real-time-ntfn-service -n chat-app

echo ""
echo "⏳ Waiting for deployments to be ready..."
kubectl rollout status deployment/message-service -n chat-app --timeout=60s
kubectl rollout status deployment/real-time-ntfn-service -n chat-app --timeout=60s

echo ""
echo "✅ Done! Services have been rebuilt and redeployed."
echo ""
echo "Check status with:"
echo "  kubectl get pods -n chat-app"

