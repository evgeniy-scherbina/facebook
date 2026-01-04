# Terraform Configuration for Kubernetes Cluster

This Terraform configuration creates a Kubernetes cluster with:
- 1 control plane node (t2.micro, Ubuntu 22.04)
- 2 worker nodes (t2.micro, Ubuntu 22.04)

## Prerequisites

1. **AWS CLI configured** with credentials
   ```bash
   aws configure
   ```

2. **Terraform installed** (>= 1.0)
   ```bash
   terraform version
   ```

## Usage

1. **Initialize Terraform**
   ```bash
   cd terraform
   terraform init
   ```

2. **Review the plan**
   ```bash
   terraform plan
   ```

3. **Apply the configuration**
   ```bash
   terraform apply
   ```

4. **View outputs**
   After applying, Terraform will display:
   - Instance IDs
   - Public IP addresses
   - Public DNS names
   - SSH commands

5. **Destroy resources**
   ```bash
   terraform destroy
   ```

## Configuration

### Variables

You can customize the configuration by creating a `terraform.tfvars` file:

```hcl
aws_region   = "us-west-2"
project_name = "my-project"
my_ip        = "174.169.160.191/32"  # Your IP for kubectl and NodePort access
```

Or set them via command line:
```bash
terraform apply -var="aws_region=us-west-2" -var="my_ip=174.169.160.191/32"
```

### Security Groups

**k8s-control-plane-sg:**
- SSH (port 22) from anywhere
- Kubernetes API Server (port 6443) from workers SG
- Kubernetes API Server (port 6443) from your IP (for kubectl)
- All outbound traffic

**k8s-workers-sg:**
- SSH (port 22) from anywhere
- Kubelet API (port 10250) from control plane SG
- NodePort services (ports 30000-32767) from your IP
- All outbound traffic

## Outputs

After `terraform apply`, you'll see:
- Control plane instance ID, IP, and DNS
- Worker instance IDs, IPs, and DNS
- SSH connection commands for all nodes
- kubectl configuration command

## Connecting to Instances

Use the SSH commands from the output, or manually:
```bash
ssh -i ~/.ssh/k8s.pem ubuntu@<public-ip>
```

**Note:** The configuration uses the existing SSH key pair named "k8s" from your AWS account. Make sure you have the corresponding private key file (`~/.ssh/k8s.pem`) on your local machine.

