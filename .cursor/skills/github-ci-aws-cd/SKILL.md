---
name: github-ci-aws-cd
description: Manages GitHub Actions CI and AWS CD for this messenger monorepo. Use when writing or changing workflows under .github/, ECR/EC2 deploy scripts, docker-compose deploy, ALB, versioning gates, or release pipelines.
---

# GitHub CI and AWS CD

You manage GitHub CI and AWS continuous delivery for this repository.

## Role

Keep CI green and CD boring: reproducible builds, explicit secrets, small blast radius, and the same artifacts local Docker and AWS consume. Prefer additive pipelines over one-off shell folklore.

## When to Apply

- Creating or editing GitHub Actions workflows
- Changing `maintenance/ecr`, `maintenance/ec2`, or `maintenance/alb` deploy paths
- Wiring version/`VERSION`, Docker images, or environment promotion
- Debugging failed CI jobs or broken AWS deploys

## This Repo Layout

| Area | Path |
|------|------|
| Client | `client/` — Vite React; `npm ci` / `npm test` / `npm run build` |
| Server | `server/` — Go module; `go test ./...` / Docker image |
| Local stack | root `docker-compose.yml` |
| ECR build/push | `maintenance/ecr/` (`build-and-push.sh`, `create-repos.sh`) |
| EC2 runtime | `maintenance/ec2/` (`deploy.sh`, compose frontend/backend) |
| ALB | `maintenance/alb/` |
| Version | root `VERSION` (shared by client/server health badge) |

There may be **no** `.github/workflows` yet — add them when introducing CI; do not invent AWS account IDs or secrets in-repo.

## CI Principles (GitHub Actions)

1. **PR checks:** client Vitest + server `go test` (scoped packages OK if full suite needs services); lint/format if already adopted.
2. **Pin tooling** — Node from repo pins / `engines`; Go from `go.mod`.
3. **Cache** — `npm` and Go module caches; avoid caching secrets or `.env`.
4. **No production deploy from fork PRs** — build/test only; deploy on protected branches with environment approvals.
5. **Artifacts** — optional Docker build on main; tag with `VERSION` + git SHA.
6. **CHANGELOG** — notable CI/CD user-facing changes belong in `CHANGELOG.md` under NEXT RELEASE (see project changelog rule).

## CD Principles (AWS)

1. **Build once** — ECR images from CI or `maintenance/ecr/build-and-push.sh`; EC2 pulls tags, does not rebuild from laptop ad hoc when CI exists.
2. **Config via env/secrets** — `DATABASE_URL`, OIDC, `CONFIG_PATH`; never commit `server/config.json` or `.env` with secrets.
3. **Compose on EC2** — prefer existing `maintenance/ec2/docker-compose*.yml` patterns over rewriting deploy topology.
4. **Health** — gate rollout on `/health` (`status` + `version`).
5. **Rollback** — redeploy previous image tag; document tag scheme (`VERSION`, SHA).
6. **Least privilege** — IAM for ECR push/pull and EC2 pull only; no long-lived keys in workflows if OIDC-to-AWS is available.

## Workflow Shape (default when adding CI)

```text
on: pull_request, push to main/dev
jobs:
  client — checkout, setup-node, npm ci, npm test, npm run build
  server — checkout, setup-go, go test ./...
  # optional: docker build (no push on PR)
```

Deploy (separate workflow or `workflow_dispatch` / tag): build-push ECR → SSH/SSM or pull on EC2 → `deploy.sh` → health check.

## Deliverables When Asked

- Concrete workflow YAML paths and job matrix
- Required GitHub secrets/variables list (names only)
- Deploy runbook steps using existing `maintenance/*` scripts
- Failure diagnosis: which job, log signature, and fix
