#!/bin/bash
set -e

# Configuration
if [ -z "$ECR_PUBLIC_ALIAS" ]; then
  ECR_PUBLIC_ALIAS=$(aws ecr-public describe-registries --region us-east-1 --query 'registries[0].aliases[0].name' --output text 2>/dev/null)
fi
ECR_PUBLIC_REGISTRY="public.ecr.aws"
PROJECT_NAME="${PROJECT_NAME:-facebook-chat}"
MAX_IMAGES_TO_KEEP="${MAX_IMAGES_TO_KEEP:-3}"  # Keep 3 most recent images

# Define services (add new services here)
SERVICES=(
  "message-service:services/message/Dockerfile"
  "real-time-ntfn-service:services/real-time-ntfn/Dockerfile"
  "sum-service:services/sum/Dockerfile"
  "mul-service:services/mul/Dockerfile"
  "calc-service:services/calc/Dockerfile"
)

echo "Building and pushing Docker images to ECR Public"
echo "Registry: ${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}"
echo "Project: ${PROJECT_NAME}"
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
  
  # Clean up old images (keep only MAX_IMAGES_TO_KEEP most recent)
  echo "  Cleaning up old images (keeping ${MAX_IMAGES_TO_KEEP} most recent)..."
  
  # Get all images with their push dates, sort by date (oldest first)
  # Format: timestamp<TAB>digest
  image_list=$(aws ecr-public describe-images \
    --repository-name "${repo_name}" \
    --region us-east-1 \
    --query 'imageDetails[*].[imagePushedAt, imageDigest]' \
    --output text 2>/dev/null | sort || echo "")
  
  if [ -n "$image_list" ] && [ "$(echo "$image_list" | wc -l)" -gt 0 ]; then
    # Count total images
    total_images=$(echo "$image_list" | wc -l)
    
    if [ "$total_images" -gt "$MAX_IMAGES_TO_KEEP" ]; then
      # Calculate how many to delete
      images_to_delete_count=$((total_images - MAX_IMAGES_TO_KEEP))
      
      # Get digests of oldest images (first images_to_delete_count lines)
      images_to_delete=$(echo "$image_list" | head -n ${images_to_delete_count} | awk '{print $2}')
      
      if [ -n "$images_to_delete" ]; then
        # Delete old images one by one
        deleted=0
        for digest in $images_to_delete; do
          if aws ecr-public batch-delete-image \
            --repository-name "${repo_name}" \
            --region us-east-1 \
            --image-ids "{\"imageDigest\":\"${digest}\"}" > /dev/null 2>&1; then
            deleted=$((deleted + 1))
          fi
        done
        
        if [ "$deleted" -gt 0 ]; then
          echo "  ✓ Deleted ${deleted} old image(s) (kept ${MAX_IMAGES_TO_KEEP} most recent)"
        fi
      fi
    else
      echo "  ✓ No cleanup needed (${total_images} image(s) total, keeping ${MAX_IMAGES_TO_KEEP})"
    fi
  fi
done

echo ""
echo "All images pushed successfully!"
