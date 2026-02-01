# Kubernetes Deployment Guide

This guide walks you through deploying the chat application to your Kubernetes cluster on AWS.

## Prerequisites

1. **AWS CLI configured** with credentials
2. **kubectl configured** to access your cluster
3. **Docker** installed and running
4. **Access to your Kubernetes cluster**

## Step 1: Build and Push Docker Images to ECR Public

Use the provided script to build and push images to AWS ECR Public (free for public repositories):

```bash
# Set your ECR Public alias (optional, defaults to 'facebook-chat')
# This is your ECR Public namespace - check it in AWS Console > ECR Public
export ECR_PUBLIC_ALIAS=facebook-chat

# Run the build and push script
./scripts/build-and-push.sh
```

This script will:
- Create ECR Public repositories if they don't exist
- Build both Docker images
- Tag and push them to ECR Public
- Output the image URLs for use in Kubernetes manifests

**Note:** ECR Public is free for public repositories. Images can be pulled without authentication.

**Manual alternative:**

```bash
# Set your ECR Public alias
ECR_PUBLIC_ALIAS=facebook-chat  # Your ECR Public namespace
ECR_PUBLIC_REGISTRY="public.ecr.aws"

# Login to ECR Public (always uses us-east-1)
aws ecr-public get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin ${ECR_PUBLIC_REGISTRY}

# Create repositories (repository name is just the name, not including alias)
aws ecr-public create-repository \
  --repository-name facebook-chat-message-service \
  --region us-east-1 || true

aws ecr-public create-repository \
  --repository-name facebook-chat-real-time-ntfn-service \
  --region us-east-1 || true

# Build and push message service
docker build -f services/message/Dockerfile -t facebook-chat-message-service:latest .
docker tag facebook-chat-message-service:latest ${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/facebook-chat-message-service:latest
docker push ${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/facebook-chat-message-service:latest

# Build and push real-time-ntfn service
docker build -f services/real-time-ntfn/Dockerfile -t facebook-chat-real-time-ntfn-service:latest .
docker tag facebook-chat-real-time-ntfn-service:latest ${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/facebook-chat-real-time-ntfn-service:latest
docker push ${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/facebook-chat-real-time-ntfn-service:latest
```

## Step 2: Update Kubernetes Manifests

Update the image references in your Kubernetes deployment files to use the ECR images.

The image format for ECR Public is:
```
public.ecr.aws/<ALIAS>/<repository-name>:latest
```

Where:
- `<ALIAS>` is your ECR Public alias/namespace (e.g., `facebook-chat`)
- `<repository-name>` is just the repository name (e.g., `facebook-chat-message-service`)

For example:
- `public.ecr.aws/facebook-chat/facebook-chat-message-service:latest`
- `public.ecr.aws/facebook-chat/facebook-chat-real-time-ntfn-service:latest`

**Important:** When creating repositories with `aws ecr-public create-repository`, use only the repository name (without the alias). The alias is part of the registry URL, not the repository name.

## Step 3: Deploy to Kubernetes

```bash
# Apply all resources
kubectl apply -k k8s/

# Or apply individually
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -k k8s/message/
kubectl apply -k k8s/real-time-ntfn/
```

## Step 4: Verify Deployment

```bash
# Check pods
kubectl get pods -n chat-app

# Check services
kubectl get svc -n chat-app

# Check deployments
kubectl get deployments -n chat-app

# View logs
kubectl logs -n chat-app -l app=message-service -f
kubectl logs -n chat-app -l app=real-time-ntfn-service -f
```

## Step 5: Access the Application

### Option 1: Port Forwarding

```bash
# Forward message service
kubectl port-forward -n chat-app svc/message-service 8080:80

# Access at http://localhost:8080
```

### Option 2: NodePort (if configured)

If your services use NodePort, you can access them via any node's IP on the NodePort.

### Option 3: Ingress (if configured)

If you have an ingress controller installed, access via the ingress hostname.

## Troubleshooting

### Images not pulling

If pods fail with `ImagePullBackOff`:
1. Verify ECR Public repository exists and is public
2. Check the image URL is correct (should start with `public.ecr.aws/`)
3. For ECR Public, no authentication is needed for pulling (images are public)
4. Verify the repository is actually public in AWS Console

### Pods not starting

```bash
# Describe a pod to see events
kubectl describe pod -n chat-app <pod-name>

# Check logs
kubectl logs -n chat-app <pod-name>
```

## Updating the Application

After making code changes:

1. Rebuild and push images:
   ```bash
   ./scripts/build-and-push.sh
   ```

2. Restart deployments to pull new images:
   ```bash
   kubectl rollout restart deployment/message-service -n chat-app
   kubectl rollout restart deployment/real-time-ntfn-service -n chat-app
   ```

   Or delete pods to force recreation:
   ```bash
   kubectl delete pods -n chat-app -l app=message-service
   kubectl delete pods -n chat-app -l app=real-time-ntfn-service
   ```

