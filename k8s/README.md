# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the chat application.

## Directory Structure

```
k8s/
├── namespace.yaml              # Creates the chat-app namespace
├── configmap.yaml              # Shared configuration for service URLs
├── ingress.yaml                # Ingress configuration (optional)
├── kustomization.yaml         # Root kustomization for all resources
├── kind-config.yaml            # Kind cluster configuration
├── message/                    # Message service manifests
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── real-time-ntfn/            # Real-time notification service manifests
    ├── deployment.yaml
    ├── service.yaml
    └── kustomization.yaml
```

## Prerequisites

1. **Kubernetes cluster** (local or remote)
2. **kubectl** configured to access your cluster
3. **Docker images** built and available:
   - `message-service:latest`
   - `real-time-ntfn-service:latest`

## Building and Loading Images (for Kind)

If using Kind (Kubernetes in Docker) for local development:

```bash
# Build Docker images
docker build -f services/message/Dockerfile -t message-service:latest .
docker build -f services/real-time-ntfn/Dockerfile -t real-time-ntfn-service:latest .

# Load images into Kind cluster
kind load docker-image message-service:latest
kind load docker-image real-time-ntfn-service:latest
```

## Deployment

### Option 1: Using kubectl (apply all files)

```bash
# Apply all manifests (recursive)
kubectl apply -f k8s/

# Or apply individually
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/message/
kubectl apply -f k8s/real-time-ntfn/
kubectl apply -f k8s/ingress.yaml  # Optional

# Or apply a specific service
kubectl apply -f k8s/message/
kubectl apply -f k8s/real-time-ntfn/
```

### Option 2: Using Kustomize

```bash
# Apply all resources
kubectl apply -k k8s/

# Or apply individual services
kubectl apply -k k8s/message/
kubectl apply -k k8s/real-time-ntfn/
```

## Accessing the Services

### Port Forwarding (Recommended for local testing)

```bash
# Forward message service
kubectl port-forward -n chat-app svc/message-service 8080:80

# Forward real-time notification service
kubectl port-forward -n chat-app svc/real-time-ntfn-service 8081:80
```

Then access the UI at `http://localhost:8080`

### Using Ingress (if configured)

If you have an ingress controller installed (e.g., NGINX Ingress):

1. Install NGINX Ingress Controller (if not already installed):
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
```

2. Add to `/etc/hosts`:
```
127.0.0.1 chat.local
```

3. Access at `http://chat.local`

## Checking Deployment Status

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

# Describe a pod for troubleshooting
kubectl describe pod -n chat-app <pod-name>
```

## Scaling

```bash
# Scale message service
kubectl scale deployment message-service -n chat-app --replicas=3

# Scale real-time notification service
kubectl scale deployment real-time-ntfn-service -n chat-app --replicas=3
```

## Cleanup

```bash
# Delete all resources
kubectl delete -f k8s/

# Or using kustomize
kubectl delete -k k8s/

# Delete namespace (will delete everything in it)
kubectl delete namespace chat-app
```

## Configuration

### Environment Variables

- `PORT` - Service port (default: 8080 for message, 8081 for real-time-ntfn)
- `NOTIFICATION_SERVICE_URL` - Set via ConfigMap (default: `http://real-time-ntfn-service`)

### Resource Limits

Default resource requests/limits:
- Memory: 64Mi request, 128Mi limit
- CPU: 100m request, 200m limit

Adjust in the deployment files as needed.

## Troubleshooting

1. **Pods not starting**: Check pod logs and events
   ```bash
   kubectl describe pod -n chat-app <pod-name>
   kubectl logs -n chat-app <pod-name>
   ```

2. **Services not communicating**: Verify service names and ports
   ```bash
   kubectl get svc -n chat-app
   kubectl exec -n chat-app <pod-name> -- nslookup real-time-ntfn-service
   ```

3. **Images not found**: Ensure images are built and loaded into the cluster
   ```bash
   # For Kind
   kind load docker-image message-service:latest
   kind load docker-image real-time-ntfn-service:latest
   ```

