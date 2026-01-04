output "control_plane_id" {
  description = "ID of the control plane instance"
  value       = aws_instance.k8s_control_plane.id
}

output "control_plane_public_ip" {
  description = "Public IP address of the control plane instance"
  value       = aws_instance.k8s_control_plane.public_ip
}

output "control_plane_public_dns" {
  description = "Public DNS name of the control plane instance"
  value       = aws_instance.k8s_control_plane.public_dns
}

output "worker_ids" {
  description = "IDs of the worker instances"
  value       = aws_instance.k8s_workers[*].id
}

output "worker_public_ips" {
  description = "Public IP addresses of the worker instances"
  value       = aws_instance.k8s_workers[*].public_ip
}

output "worker_public_dns" {
  description = "Public DNS names of the worker instances"
  value       = aws_instance.k8s_workers[*].public_dns
}

output "ssh_control_plane" {
  description = "SSH command to connect to control plane"
  value       = "ssh -i ~/.ssh/k8s.pem ubuntu@${aws_instance.k8s_control_plane.public_ip}"
}

output "ssh_workers" {
  description = "SSH commands to connect to worker nodes"
  value = [
    for i, instance in aws_instance.k8s_workers :
    "ssh -i ~/.ssh/k8s.pem ubuntu@${instance.public_ip}"
  ]
}

output "kubectl_config" {
  description = "Command to configure kubectl (run after setting up Kubernetes)"
  value       = "export KUBECONFIG=~/.kube/config && kubectl config set-cluster k8s-cluster --server=https://${aws_instance.k8s_control_plane.public_ip}:6443"
}

