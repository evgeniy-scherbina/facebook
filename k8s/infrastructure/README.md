# Infrastructure Components

This directory contains Kubernetes infrastructure components that need to be installed separately from the application.

## Components

### Ingress Controller (NGINX)

The NGINX Ingress Controller provides HTTP/HTTPS routing to services in the cluster.

**Installation:**
```bash
kubectl apply -f k8s/infrastructure/ingress-nginx/deploy.yaml
```

Or use the installation script:
```bash
./scripts/install-infrastructure.sh
```

**Verify installation:**
```bash
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
```

**Get external IP:**
```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

**Version:** v1.11.1

**Note:** This is a one-time installation. The Ingress Controller persists across application deployments.

