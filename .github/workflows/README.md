# GitHub Actions Workflows

## Build and Push Docker Images

This workflow automatically builds and pushes Docker images to AWS ECR Public on every push to `master` or `main` branch.

### Setup

1. **Add AWS credentials as GitHub Secrets:**
   - Go to your repository Settings > Secrets and variables > Actions
   - Add the following secrets:
     - `AWS_ACCESS_KEY_ID` - Your AWS access key ID
     - `AWS_SECRET_ACCESS_KEY` - Your AWS secret access key

2. **Optional environment variables:**
   - `PROJECT_NAME` - Defaults to `facebook-chat`
   - `ECR_PUBLIC_ALIAS` - Auto-detected, but can be overridden

### What it does

1. Checks out the code
2. Configures AWS credentials
3. Sets up Docker Buildx
4. Runs `scripts/build-and-push.sh` to build and push all services
5. Outputs the image URLs in the workflow summary

### Manual trigger

You can also manually trigger the workflow:
- Go to Actions tab
- Select "Build and Push Docker Images"
- Click "Run workflow"

