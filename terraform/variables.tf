variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name for resource naming"
  type        = string
  default     = "facebook-chat"
}

variable "my_ip" {
  description = "Your IP address for kubectl and NodePort access (CIDR format, e.g., 174.169.160.191/32)"
  type        = string
  default     = "174.169.160.191/32"
}

