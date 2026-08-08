# Containerized runner for the "GitHub ticket processing" cycle described in
# CLAUDE.md: fetches the repo, runs `claude -p` in a loop (see entrypoint.sh),
# and dispatches per-ticket agents. Build context is the repo root:
#
#   docker build -f maintenance/claude-runner/Dockerfile -t claude-runner .
#
# Go toolchain is copied from the same golang image tag server/Dockerfile's
# build stage uses, so `go build`/`go vet`/`go test` behave identically here.
FROM golang:1.25-bookworm AS go-toolchain

FROM node:22-bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		git \
		curl \
		ca-certificates \
		docker.io \
		awscli \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=go-toolchain /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN npm install -g @anthropic-ai/claude-code

# Non-root user, matching server/Dockerfile's adduser pattern. The docker
# group lets this user talk to the host's mounted /var/run/docker.sock
# (Docker-outside-of-Docker — see docker-compose.yml) without running the
# container itself as root.
RUN groupadd -f docker \
	&& useradd --create-home --shell /bin/bash --groups docker claude \
	&& mkdir -p /workspace \
	&& chown -R claude:claude /workspace /home/claude

COPY --chown=claude:claude maintenance/claude-runner/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

USER claude
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
