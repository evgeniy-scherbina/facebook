# Terraform Configuration for EC2 Instances

This Terraform configuration creates 3 EC2 t2.micro instances running Ubuntu 22.04.

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
```

Or set them via command line:
```bash
terraform apply -var="aws_region=us-west-2"
```

### Security Group

The default security group allows:
- SSH (port 22) from anywhere
- HTTP (port 80) from anywhere
- HTTPS (port 443) from anywhere
- All outbound traffic

**Note:** For production, restrict SSH access to your IP address.

## Outputs

After `terraform apply`, you'll see:
- Instance IDs
- Public IPs
- Public DNS names
- SSH connection commands

## Connecting to Instances

Use the SSH commands from the output, or manually:
```bash
ssh -i ~/.ssh/your-key.pem ubuntu@<public-ip>
```

**Note:** You'll need to specify your SSH key. If you don't have one, you can add a `key_name` parameter to the `aws_instance` resource.

