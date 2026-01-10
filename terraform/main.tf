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

  # SSH access
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # API Server access from my IP for kubectl
  ingress {
    description = "Kubernetes API Server (kubectl) from my IP"
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = ["${var.my_ip}/32"]
  }

  # Allow all outbound traffic
  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "k8s-control-plane-sg"
  }
}

# Security group for Kubernetes worker nodes
resource "aws_security_group" "k8s_workers_sg" {
  name        = "k8s-workers-sg"
  description = "Security group for Kubernetes worker nodes"

  # SSH access
  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # NodePort services from my IP
  ingress {
    description = "NodePort services from my IP"
    from_port   = 30000
    to_port     = 32767
    protocol    = "tcp"
    cidr_blocks = ["${var.my_ip}/32"]
  }

  # Allow all outbound traffic
  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "k8s-workers-sg"
  }
}

# Security group rules for cross-references (to break circular dependency)

# Allow API Server access from workers to control plane
resource "aws_security_group_rule" "control_plane_api_from_workers" {
  type                     = "ingress"
  from_port                = 6443
  to_port                  = 6443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.k8s_workers_sg.id
  security_group_id        = aws_security_group.k8s_control_plane_sg.id
  description              = "Kubernetes API Server from workers"
}

# Allow Kubelet API access from control plane to workers
resource "aws_security_group_rule" "workers_kubelet_from_control_plane" {
  type                     = "ingress"
  from_port                = 10250
  to_port                  = 10250
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.k8s_control_plane_sg.id
  security_group_id        = aws_security_group.k8s_workers_sg.id
  description              = "Kubelet API from control plane"
}

# Create Kubernetes control plane instance
resource "aws_instance" "k8s_control_plane" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = "t2.micro"
  key_name      = "k8s"  # Existing SSH key pair in AWS account
  
  vpc_security_group_ids = [aws_security_group.k8s_control_plane_sg.id]
  
  tags = {
    Name = "${var.project_name}-control-plane"
    Role = "control-plane"
  }

  # Optional: Add user data for initial setup
  user_data = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y curl wget git
              EOF
}

# Create Kubernetes worker instances
resource "aws_instance" "k8s_workers" {
  count         = 2
  ami           = data.aws_ami.ubuntu.id
  instance_type  = "t2.micro"
  key_name      = "k8s"  # Existing SSH key pair in AWS account
  
  vpc_security_group_ids = [aws_security_group.k8s_workers_sg.id]
  
  tags = {
    Name = "${var.project_name}-worker-${count.index + 1}"
    Role = "worker"
  }

  # Optional: Add user data for initial setup
  user_data = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y curl wget git
              EOF
}

