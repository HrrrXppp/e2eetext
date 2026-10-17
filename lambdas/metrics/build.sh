#!/usr/bin/env bash
# Builds the Lambda deployment package (provided.al2023, arm64) at
# lambdas/metrics/build/metrics.zip. Terraform (terraform/modules/metrics)
# deploys that zip.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
mkdir -p build
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -tags lambda.norpc -o build/bootstrap .
# -FS drops stale entries from an existing zip, so no cleanup (rm) is needed.
# Fixed mtime + no extra attrs so the zip hash is reproducible.
touch -d '2000-01-01 00:00:00' build/bootstrap
(cd build && zip -X -q -FS metrics.zip bootstrap)
echo "built $(pwd)/build/metrics.zip"
