output "instance_ids" {
  description = "IDs of the EC2 instances"
  value       = aws_instance.ubuntu_instances[*].id
}

output "instance_public_ips" {
  description = "Public IP addresses of the EC2 instances"
  value       = aws_instance.ubuntu_instances[*].public_ip
}

output "instance_public_dns" {
  description = "Public DNS names of the EC2 instances"
  value       = aws_instance.ubuntu_instances[*].public_dns
}

output "ssh_commands" {
  description = "SSH commands to connect to instances"
  value = [
    for i, instance in aws_instance.ubuntu_instances :
    "ssh -i ~/.ssh/k8s.pem ubuntu@${instance.public_ip}"
  ]
}

