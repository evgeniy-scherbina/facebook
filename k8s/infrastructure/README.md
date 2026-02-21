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

### AWS Cloud Controller Manager (CCM) and LoadBalancer

For the Ingress Controller’s `LoadBalancer` Service to get an external ELB and **automatically register cluster instances**, the cluster must use the **external cloud provider**:

- **kubelet**, **kube-apiserver**, and **kube-controller-manager** must run with `--cloud-provider=external`.
- The AWS CCM then sets each node’s `spec.providerID` and registers those instance IDs with the ELB.

The Ansible playbook configures this via kubeadm (init and join configs with `cloud-provider=external`). After a full cluster recreate (Terraform + Ansible), the LoadBalancer should receive registered instances without manual steps.

