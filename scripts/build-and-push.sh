#!/bin/bash
set -e

# Configuration
if [ -z "$ECR_PUBLIC_ALIAS" ]; then
  ECR_PUBLIC_ALIAS=$(aws ecr-public describe-registries --region us-east-1 --query 'registries[0].aliases[0].name' --output text 2>/dev/null)
fi
ECR_PUBLIC_REGISTRY="public.ecr.aws"
PROJECT_NAME="${PROJECT_NAME:-facebook-chat}"

# Define services (add new services here)
SERVICES=(
  "message-service:services/message/Dockerfile"
  "real-time-ntfn-service:services/real-time-ntfn/Dockerfile"
)

echo "Building and pushing Docker images to ECR Public"
echo "Registry: ${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}"
echo ""

# Login to ECR Public
aws ecr-public get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin ${ECR_PUBLIC_REGISTRY} > /dev/null 2>&1

# Process each service
for service_config in "${SERVICES[@]}"; do
  IFS=':' read -r service_name dockerfile <<< "$service_config"
  repo_name="${PROJECT_NAME}-${service_name}"
  image_tag="${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/${repo_name}:latest"
  
  echo "Processing ${service_name}..."
  
  # Create repository if it doesn't exist
  if ! aws ecr-public describe-repositories --repository-name "${repo_name}" --region us-east-1 > /dev/null 2>&1; then
    aws ecr-public create-repository \
      --repository-name "${repo_name}" \
      --region us-east-1 \
      --catalog-data "{\"description\":\"${service_name} for ${PROJECT_NAME}\",\"architectures\":[\"amd64\"],\"operatingSystems\":[\"linux\"]}" > /dev/null 2>&1
  fi
  
  # Build image
  docker build -f "${dockerfile}" -t "${repo_name}:latest" . > /dev/null 2>&1
  docker tag "${repo_name}:latest" "${image_tag}" > /dev/null 2>&1
  
  # Push image
  docker push "${image_tag}" > /dev/null 2>&1
  
  echo "  ✓ ${image_tag}"
done

echo ""
echo "All images pushed successfully!"
