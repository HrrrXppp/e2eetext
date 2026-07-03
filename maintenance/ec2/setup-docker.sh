#!/usr/bin/env bash
set -euo pipefail

# Install Docker on Amazon Linux 2023 / Ubuntu (run once on a fresh EC2).
# Usage: sudo bash maintenance/ec2/setup-docker.sh

if command -v docker >/dev/null 2>&1; then
  echo "Docker already installed: $(docker --version)"
  exit 0
fi

if [ -f /etc/os-release ]; then
  # shellcheck source=/dev/null
  . /etc/os-release
else
  echo "Unsupported OS"
  exit 1
fi

case "${ID:-}" in
  amzn)
    dnf update -y
    dnf install -y docker
    systemctl enable --now docker
    usermod -aG docker "${SUDO_USER:-ec2-user}" || true
    ;;
  ubuntu)
    apt-get update
    apt-get install -y ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
      $(. /etc/os-release && echo "${VERSION_CODENAME}") stable" \
      > /etc/apt/sources.list.d/docker.list
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    usermod -aG docker "${SUDO_USER:-ubuntu}" || true
    ;;
  *)
    echo "Unsupported OS: ${ID}. Install Docker manually."
    exit 1
    ;;
esac

if ! docker compose version >/dev/null 2>&1; then
  echo "Installing Docker Compose plugin..."
  mkdir -p /usr/local/lib/docker/cli-plugins
  curl -SL "https://github.com/docker/compose/releases/latest/download/docker-compose-linux-$(uname -m)" \
    -o /usr/local/lib/docker/cli-plugins/docker-compose
  chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
fi

echo "Done. Log out and back in, then: docker compose version"
