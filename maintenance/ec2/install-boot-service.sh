#!/usr/bin/env bash
# Install (or refresh) the systemd unit that runs deploy.sh on every boot.
# Safe to re-run. Used manually or by Terraform via SSM.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
UNIT_SRC="${UNIT_SRC:-$ROOT_DIR/maintenance/ec2/e2eetext.service}"
UNIT_DST=/etc/systemd/system/e2eetext.service
DEPLOY_ENV_DIR=/etc/e2eetext
DEPLOY_ENV_FILE="$DEPLOY_ENV_DIR/deploy.env"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Run as root (sudo)." >&2
  exit 1
fi

if [[ ! -f "$UNIT_SRC" ]]; then
  echo "Missing unit template: $UNIT_SRC" >&2
  exit 1
fi

if [[ ! -x "$(command -v docker)" ]]; then
  echo "Docker is required before enabling the boot unit." >&2
  exit 1
fi

# Rewrite WorkingDirectory / ExecStart to this checkout. IMAGE_TAG (and
# friends) stay in maintenance/ec2/.env unless /etc/e2eetext/deploy.env is
# present (written by Terraform or by hand).
sed \
  -e "s|^WorkingDirectory=.*|WorkingDirectory=$ROOT_DIR|" \
  -e "s|^ExecStart=.*|ExecStart=/bin/bash $ROOT_DIR/maintenance/ec2/deploy.sh|" \
  "$UNIT_SRC" >"$UNIT_DST"

mkdir -p "$DEPLOY_ENV_DIR"
if [[ -n "${IMAGE_TAG:-}" ]]; then
  # Only rewrite IMAGE_TAG; leave any other keys in deploy.env alone.
  if [[ -f "$DEPLOY_ENV_FILE" ]] && grep -q '^IMAGE_TAG=' "$DEPLOY_ENV_FILE"; then
    sed -i "s|^IMAGE_TAG=.*|IMAGE_TAG=$IMAGE_TAG|" "$DEPLOY_ENV_FILE"
  elif [[ -f "$DEPLOY_ENV_FILE" ]]; then
    printf 'IMAGE_TAG=%s\n' "$IMAGE_TAG" >>"$DEPLOY_ENV_FILE"
  else
    printf 'IMAGE_TAG=%s\n' "$IMAGE_TAG" >"$DEPLOY_ENV_FILE"
  fi
  chmod 0644 "$DEPLOY_ENV_FILE"

  # Keep maintenance/ec2/.env in sync so compose/deploy see the pin even when
  # an older deploy.sh prefers an exported shell IMAGE_TAG from sourcing .env.
  ENV_DOT="$ROOT_DIR/maintenance/ec2/.env"
  if [[ -f "$ENV_DOT" ]]; then
    if grep -q '^IMAGE_TAG=' "$ENV_DOT"; then
      sed -i "s|^IMAGE_TAG=.*|IMAGE_TAG=$IMAGE_TAG|" "$ENV_DOT"
    else
      printf 'IMAGE_TAG=%s\n' "$IMAGE_TAG" >>"$ENV_DOT"
    fi
  fi
fi

systemctl daemon-reload
systemctl enable e2eetext.service

if [[ "${START_NOW:-1}" == "1" ]]; then
  systemctl restart e2eetext.service
  systemctl --no-pager --full status e2eetext.service || true
else
  echo "Unit enabled. It will run on next boot (START_NOW=0)."
fi
