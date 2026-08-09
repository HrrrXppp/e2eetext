#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/maintenance/ec2/.env}"
COMPOSE_FILE="$ROOT_DIR/maintenance/ec2/docker-compose.yml"
CONFIG_FILE="$ROOT_DIR/maintenance/ec2/config/config.json"
INSTANCE_CONFIG_FILE="$ROOT_DIR/maintenance/ec2/config/instance.json"
LEGACY_CONFIG="$ROOT_DIR/server/config.json"

compose() {
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

require_env() {
  # shellcheck source=/dev/null
  source "$ENV_FILE"
  local missing=0
  for key in DATABASE_URL AWS_ACCOUNT_ID AWS_REGION IMAGE_TAG; do
    if [[ -z "${!key:-}" ]]; then
      echo "Missing $key in $ENV_FILE" >&2
      missing=1
    fi
  done
  if [[ "$missing" -ne 0 ]]; then
    exit 1
  fi
}

preflight() {
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "Missing $ENV_FILE — copy from .env.example and edit." >&2
    exit 1
  fi

  require_env

  if [[ -d "$LEGACY_CONFIG" ]]; then
    echo "Found a directory at $LEGACY_CONFIG (Docker created it when the file was missing)." >&2
    echo "Remove it: rm -rf $LEGACY_CONFIG" >&2
    exit 1
  fi

  if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Missing $CONFIG_FILE" >&2
    echo "Run: cp maintenance/ec2/config/config.example.json maintenance/ec2/config/config.json" >&2
    echo "Then add your Google OAuth client_id and client_secret." >&2
    exit 1
  fi

  if [[ ! -f "$INSTANCE_CONFIG_FILE" ]]; then
    echo "Missing $INSTANCE_CONFIG_FILE" >&2
    echo "Production: cp maintenance/ec2/config/instance.production.example.json maintenance/ec2/config/instance.json" >&2
    echo "Development: cp maintenance/ec2/config/instance.development.example.json maintenance/ec2/config/instance.json" >&2
    exit 1
  fi

  if ! command -v aws >/dev/null 2>&1; then
    echo "AWS CLI is required on EC2 to log in to ECR." >&2
    exit 1
  fi

  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is required." >&2
    exit 1
  fi
}

show_server_logs() {
  echo ""
  echo "=== server logs (last 80 lines) ==="
  compose logs --no-color --tail=80 server || true
  echo ""
  echo "=== container status ==="
  compose ps -a || true
}

wait_for_server() {
  local attempts=60
  local i id status health

  for ((i = 1; i <= attempts; i++)); do
    id="$(compose ps -q server || true)"
    if [[ -z "$id" ]]; then
      echo "Server container is missing." >&2
      show_server_logs
      return 1
    fi

    status="$(docker inspect --format='{{.State.Status}}' "$id")"
    if [[ "$status" != "running" ]]; then
      echo "Server container status: $status" >&2
      show_server_logs
      return 1
    fi

    health="$(docker inspect --format='{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$id")"
    case "$health" in
      healthy)
        echo "Server is healthy."
        return 0
        ;;
      unhealthy)
        echo "Server health check failed." >&2
        show_server_logs
        return 1
        ;;
    esac

    sleep 5
  done

  echo "Server did not become healthy within $((attempts * 5)) seconds." >&2
  show_server_logs
  return 1
}

preflight

bash "$ROOT_DIR/maintenance/ec2/ecr-login.sh"

compose pull
if ! compose up -d; then
  show_server_logs
  exit 1
fi

if ! wait_for_server; then
  echo ""
  echo "Common fixes:"
  echo "  1. DATABASE_URL — RDS host, user, password, sslmode=require"
  echo "  2. RDS security group — allow TCP 5432 from this EC2 security group"
  echo "  3. maintenance/ec2/config/config.json — valid OAuth JSON"
  echo "  4. ECR image — build with DOCKER_PLATFORM=linux/amd64 on Apple Silicon"
  exit 1
fi

echo "Deployed. Check:"
echo "  docker compose --env-file $ENV_FILE -f $COMPOSE_FILE ps"
