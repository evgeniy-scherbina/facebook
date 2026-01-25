#!/bin/bash
set -e

# Configuration
# Auto-detect ECR Public alias, or use provided one
if [ -z "$ECR_PUBLIC_ALIAS" ]; then
  ECR_PUBLIC_ALIAS=$(aws ecr-public describe-registries --region us-east-1 --query 'registries[0].aliases[0].name' --output text 2>/dev/null || echo "facebook-chat")
fi
ECR_PUBLIC_REGISTRY="public.ecr.aws"
PROJECT_NAME="${PROJECT_NAME:-facebook-chat}"

echo "Detected ECR Public alias: ${ECR_PUBLIC_ALIAS}"

# Repository names (just the name, without alias)
MESSAGE_REPO="${PROJECT_NAME}-message-service"
REAL_TIME_REPO="${PROJECT_NAME}-real-time-ntfn-service"

# Full image tags (alias is part of the registry URL)
MESSAGE_TAG="${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/${MESSAGE_REPO}:latest"
REAL_TIME_TAG="${ECR_PUBLIC_REGISTRY}/${ECR_PUBLIC_ALIAS}/${REAL_TIME_REPO}:latest"

echo ""
echo "Building and pushing Docker images to ECR Public (free for public repos)..."
echo "Registry: ${ECR_PUBLIC_REGISTRY}"
echo "Using ECR Public alias: ${ECR_PUBLIC_ALIAS}"
echo ""

# Login to ECR Public
echo "Logging in to ECR Public..."
aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin ${ECR_PUBLIC_REGISTRY}

# Create ECR Public repositories if they don't exist
echo "Creating ECR Public repositories..."
echo "Repository name: ${MESSAGE_REPO}"
echo "Alias: ${ECR_PUBLIC_ALIAS}"

# Check if repository exists
if aws ecr-public describe-repositories --repository-name ${MESSAGE_REPO} --region us-east-1 2>/dev/null; then
  echo "Repository ${MESSAGE_REPO} already exists"
else
  echo "Creating repository ${MESSAGE_REPO}..."
  aws ecr-public create-repository \
    --repository-name ${MESSAGE_REPO} \
    --region us-east-1 \
    --catalog-data "{\"description\":\"Message service for Facebook chat application\",\"architectures\":[\"amd64\"],\"operatingSystems\":[\"linux\"]}"
  
  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to create repository ${MESSAGE_REPO}"
    echo "Please check:"
    echo "1. Your ECR Public alias is correct: ${ECR_PUBLIC_ALIAS}"
    echo "2. You have permissions to create repositories"
    echo "3. The repository name is valid"
    exit 1
  fi
  echo "Repository ${MESSAGE_REPO} created successfully"
fi

echo ""
echo "Repository name: ${REAL_TIME_REPO}"
if aws ecr-public describe-repositories --repository-name ${REAL_TIME_REPO} --region us-east-1 2>/dev/null; then
  echo "Repository ${REAL_TIME_REPO} already exists"
else
  echo "Creating repository ${REAL_TIME_REPO}..."
  aws ecr-public create-repository \
    --repository-name ${REAL_TIME_REPO} \
    --region us-east-1 \
    --catalog-data "{\"description\":\"Real-time notification service for Facebook chat application\",\"architectures\":[\"amd64\"],\"operatingSystems\":[\"linux\"]}"
  
  if [ $? -ne 0 ]; then
    echo "ERROR: Failed to create repository ${REAL_TIME_REPO}"
    exit 1
  fi
  echo "Repository ${REAL_TIME_REPO} created successfully"
fi

echo ""
echo "Verifying repositories exist..."
aws ecr-public describe-repositories --repository-names ${MESSAGE_REPO} ${REAL_TIME_REPO} --region us-east-1

# Build and push message service
echo "Building message-service..."
docker build -f services/message/Dockerfile -t ${MESSAGE_REPO}:latest .
docker tag ${MESSAGE_REPO}:latest ${MESSAGE_TAG}

echo "Pushing message-service to ECR Public..."
docker push ${MESSAGE_TAG}

# Build and push real-time-ntfn service
echo "Building real-time-ntfn-service..."
docker build -f services/real-time-ntfn/Dockerfile -t ${REAL_TIME_REPO}:latest .
docker tag ${REAL_TIME_REPO}:latest ${REAL_TIME_TAG}

echo "Pushing real-time-ntfn-service to ECR..."
docker push ${REAL_TIME_TAG}

echo ""
echo "Images pushed successfully to ECR Public!"
echo "Message service: ${MESSAGE_TAG}"
echo "Real-time-ntfn service: ${REAL_TIME_TAG}"
echo ""
echo "These images are PUBLIC and can be pulled without authentication."
echo "Update your Kubernetes deployments to use these images."
echo ""
echo "Note: Make sure your ECR Public alias '${ECR_PUBLIC_ALIAS}' is correct."
echo "You can find it in AWS Console > ECR Public > Repositories"

