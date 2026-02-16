# AWS Cloud Controller Manager

The AWS Cloud Controller Manager (CCM) enables Kubernetes to interact with AWS services, particularly for provisioning LoadBalancers.

## Installation

CCM is automatically installed as part of cluster setup via Ansible. To install manually:

```bash
kubectl apply -f k8s/infrastructure/aws-ccm/rbac.yaml
kubectl apply -f k8s/infrastructure/aws-ccm/daemonset.yaml
```

## Prerequisites

1. **IAM Permissions**: Nodes need IAM permissions to manage LoadBalancers and EC2 resources. These are configured in Terraform and attached via instance profiles.

2. **Node Tags**: Nodes should be tagged with:
   - `kubernetes.io/cluster/kubernetes=owned` (or `shared`)

3. **Subnet Tags** (optional, for public LoadBalancers):
   - `kubernetes.io/role/elb=1` for public subnets
   - `kubernetes.io/role/internal-elb=1` for internal subnets

## Verification

```bash
# Check CCM pods
kubectl get pods -n kube-system -l app=aws-cloud-controller-manager

# Check CCM logs
kubectl logs -n kube-system -l app=aws-cloud-controller-manager
```

## After Installation

Once CCM is running, LoadBalancer services should automatically provision AWS LoadBalancers. Check with:

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

The `EXTERNAL-IP` should change from `<pending>` to an actual IP/hostname.

