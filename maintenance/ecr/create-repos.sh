#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

IMAGE_TAG="${IMAGE_TAG:-latest}"

echo "Logging in to ECR: $ECR_REGISTRY"
aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "$ECR_REGISTRY"

for repo in e2eetext-server e2eetext-client; do
  if aws ecr describe-repositories --repository-names "$repo" --region "$AWS_REGION" >/dev/null 2>&1; then
    echo "Repository exists: $repo"
  else
    echo "Creating repository: $repo"
    aws ecr create-repository \
      --repository-name "$repo" \
      --region "$AWS_REGION" \
      --image-scanning-configuration scanOnPush=true \
      --encryption-configuration encryptionType=AES256
  fi
done

echo "ECR repositories ready."
