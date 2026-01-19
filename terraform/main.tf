terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Get the latest Ubuntu 22.04 AMI
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# Security group for Kubernetes control plane
resource "aws_security_group" "k8s_control_plane_sg" {
  name        = "k8s-control-plane-sg"
  description = "Security group for Kubernetes control plane node"

  tags = {
    Name = "k8s-control-plane-sg"
  }
}

# Security group for Kubernetes worker nodes
resource "aws_security_group" "k8s_workers_sg" {
  name        = "k8s-workers-sg"
  description = "Security group for Kubernetes worker nodes"

  tags = {
    Name = "k8s-workers-sg"
  }
}

# Security group rules - all defined as separate resources

# Control Plane Rules
resource "aws_security_group_rule" "control_plane_ssh" {
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.k8s_control_plane_sg.id
  description       = "SSH access"
}

resource "aws_security_group_rule" "control_plane_api_from_my_ip" {
  type              = "ingress"
  from_port         = 6443
  to_port           = 6443
  protocol          = "tcp"
  cidr_blocks       = ["${var.my_ip}/32"]
  security_group_id = aws_security_group.k8s_control_plane_sg.id
  description       = "Kubernetes API Server (kubectl) from my IP"
}

resource "aws_security_group_rule" "control_plane_api_from_workers" {
  type                     = "ingress"
  from_port                = 6443
  to_port                  = 6443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.k8s_workers_sg.id
  security_group_id        = aws_security_group.k8s_control_plane_sg.id
  description              = "Kubernetes API Server from workers"
}

resource "aws_security_group_rule" "control_plane_egress_all" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.k8s_control_plane_sg.id
  description       = "Allow all outbound traffic"
}

# Worker Rules
resource "aws_security_group_rule" "workers_ssh" {
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.k8s_workers_sg.id
  description       = "SSH access"
}

resource "aws_security_group_rule" "workers_nodeport_from_my_ip" {
  type              = "ingress"
  from_port         = 30000
  to_port           = 32767
  protocol          = "tcp"
  cidr_blocks       = ["${var.my_ip}/32"]
  security_group_id = aws_security_group.k8s_workers_sg.id
  description       = "NodePort services from my IP"
}

resource "aws_security_group_rule" "workers_kubelet_from_control_plane" {
  type                     = "ingress"
  from_port                = 10250
  to_port                  = 10250
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.k8s_control_plane_sg.id
  security_group_id        = aws_security_group.k8s_workers_sg.id
  description              = "Kubelet API from control plane"
}

resource "aws_security_group_rule" "workers_egress_all" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.k8s_workers_sg.id
  description       = "Allow all outbound traffic"
}

# Allow HTTP server access from workers to control plane (for fetching kubeconfig and join command)
resource "aws_security_group_rule" "control_plane_http_from_workers" {
  type                     = "ingress"
  from_port                = 8080
  to_port                  = 8080
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.k8s_workers_sg.id
  security_group_id        = aws_security_group.k8s_control_plane_sg.id
  description              = "HTTP server for kubeconfig and join command from workers"
}

# Create Kubernetes control plane instance
resource "aws_instance" "k8s_control_plane" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t3.medium"
  key_name      = "k8s"  # Existing SSH key pair in AWS account
  
  vpc_security_group_ids = [aws_security_group.k8s_control_plane_sg.id]
  
  tags = {
    Name = "${var.project_name}-control-plane"
    Role = "control-plane"
  }

  # User data script to install Ansible and run playbook
  user_data = <<-EOF
              #!/bin/bash
              set -e
              
              # Install Ansible and git
              apt-get update
              apt-get install -y python3 python3-pip python3-apt curl git
              pip3 install ansible
              
              # Clone repository as ubuntu user
              su - ubuntu -c "git clone https://github.com/evgeniy-scherbina/facebook.git /home/ubuntu/facebook"
              
              # Run playbook as ubuntu user with control_plane=true for control plane node
              # Redirect output to log file for visibility
              su - ubuntu -c "cd /home/ubuntu/facebook/ansible && ansible-playbook playbook.yml -e 'control_plane=true' > /home/ubuntu/ansible-playbook.log 2>&1"
              EOF
}

# Create Kubernetes worker instances
resource "aws_instance" "k8s_workers" {
  count         = 1 # TODO: set to 2
  ami           = data.aws_ami.ubuntu.id
  instance_type  = "t3.small"
  key_name      = "k8s"  # Existing SSH key pair in AWS account
  
  vpc_security_group_ids = [aws_security_group.k8s_workers_sg.id]
  
  tags = {
    Name = "${var.project_name}-worker-${count.index + 1}"
    Role = "worker"
  }

  # User data script to install Ansible and run playbook
  user_data = <<-EOF
              #!/bin/bash
              set -e
              
              # Install Ansible and git
              apt-get update
              apt-get install -y python3 python3-pip python3-apt curl git
              pip3 install ansible
              
              # Clone repository as ubuntu user
              su - ubuntu -c "git clone https://github.com/evgeniy-scherbina/facebook.git /home/ubuntu/facebook"
              
              # Run playbook as ubuntu user (worker node - control_plane defaults to false)
              # Redirect output to log file for visibility
              su - ubuntu -c "cd /home/ubuntu/facebook/ansible && ansible-playbook playbook.yml > /home/ubuntu/ansible-playbook.log 2>&1"
              EOF
}

