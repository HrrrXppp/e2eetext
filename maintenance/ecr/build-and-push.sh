#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=lib.sh
source "$SCRIPT_DIR/lib.sh"

IMAGE_TAG="${IMAGE_TAG:-latest}"
# The client is also published as static files (deploy-client.yml, #28), so
# build-images.yml sets BUILD_CLIENT=false. Local runs still build both.
BUILD_CLIENT="${BUILD_CLIENT:-true}"
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
docker build "${BUILD_ARGS[@]}" -f "$ROOT_DIR/server/Dockerfile" -t "$SERVER_IMAGE" "$ROOT_DIR"
docker push "$SERVER_IMAGE"

if [[ "$BUILD_CLIENT" == "true" ]]; then
  echo "Building client -> $CLIENT_IMAGE"
  docker build "${BUILD_ARGS[@]}" -f "$ROOT_DIR/client/Dockerfile" -t "$CLIENT_IMAGE" "$ROOT_DIR"
  docker push "$CLIENT_IMAGE"
fi

echo ""
echo "Pushed:"
echo "  $SERVER_IMAGE"
if [[ "$BUILD_CLIENT" == "true" ]]; then
  echo "  $CLIENT_IMAGE"
fi
echo ""
echo "On EC2, set in maintenance/ec2/.env:"
echo "  AWS_ACCOUNT_ID=$AWS_ACCOUNT_ID"
echo "  AWS_REGION=$AWS_REGION"
echo "  IMAGE_TAG=$IMAGE_TAG"
