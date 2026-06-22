#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

IMAGE_TAG="${IMAGE_TAG:-latest}"
DOCKER_PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"
BUILD_ARGS=()
if [[ -n "$DOCKER_PLATFORM" ]]; then
  BUILD_ARGS=(--platform "$DOCKER_PLATFORM")
fi

echo "Logging in to ECR: $ECR_REGISTRY"
aws ecr get-login-password --region "$AWS_REGION" \
  | docker login --username AWS --password-stdin "$ECR_REGISTRY"

SERVER_IMAGE="${ECR_REGISTRY}/e2eetext-server:${IMAGE_TAG}"
CLIENT_IMAGE="${ECR_REGISTRY}/e2eetext-client:${IMAGE_TAG}"

echo "Building server -> $SERVER_IMAGE"
docker build "${BUILD_ARGS[@]}" -t "$SERVER_IMAGE" "$ROOT_DIR/server"
docker push "$SERVER_IMAGE"

echo "Building client -> $CLIENT_IMAGE"
docker build "${BUILD_ARGS[@]}" -t "$CLIENT_IMAGE" "$ROOT_DIR/client"
docker push "$CLIENT_IMAGE"

echo ""
echo "Pushed:"
echo "  $SERVER_IMAGE"
echo "  $CLIENT_IMAGE"
echo ""
echo "On EC2, set in maintenance/ec2/.env:"
echo "  AWS_ACCOUNT_ID=$AWS_ACCOUNT_ID"
echo "  AWS_REGION=$AWS_REGION"
echo "  IMAGE_TAG=$IMAGE_TAG"
