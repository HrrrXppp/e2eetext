# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Release version is defined in [`VERSION`](./VERSION).

On branches that are not yet merged to `main`, use **`NEXT RELEASE`** as the changelog heading. When a release is cut on `main`, rename that section to the semver from `VERSION` and add the release date.

## [NEXT RELEASE]

Pre-MVP alpha. New chats and messages use hybrid ML-KEM-768 + X25519 end-to-end encryption; the server stores only opaque ciphertext and key wraps.

### Added

#### Server

- Go HTTP API (`/api/v1`) with PostgreSQL (pgx) and SQL migrations
- Federated node-scoped resource IDs
- OIDC authentication with request-derived redirect URLs (Google and Apple seeded by default)
- Generic `private_key_jwt` client authentication (RFC 7523 / OpenID Connect Core §9) for providers with no static client secret (e.g. Apple); per-provider scopes, `response_mode`, and client-authentication strategy are now data on the `oidc_providers` row (migration `000007`), not hardcoded per-provider Go branches
- Dual `GET`/`POST` OIDC callback route so `response_mode=form_post` providers (Apple) work; one-time out-of-band display names (Apple's first-sign-in-only `user` payload) flow through to user registration
- User registration, search, and profile API
- Chats, messages, unread tracking, and a WebSocket event hub
- WebSocket endpoint and application route wiring
- `GET /health` returns `status` and `version`
- E2EE identity-key and chat key-wrap APIs; chat create requires per-member wraps
- Disappearing messages: chat-level TTL (`disappear_after_minutes`, default 60 days) using `messages.created_at`; `pg_cron` schedules `purge_expired_messages()` in migration `000006`
- Real integration test suite (`server/internal/integration`, `go test -tags=integration`) exercising real Postgres, real HTTP handlers, and real OIDC auth verification against a mock identity provider

#### Client

- React 19 + TypeScript + Vite SPA
- OIDC sign-in, session management, and landing page
- Chats UI, messaging, and real-time WebSocket updates
- Deploy-time dev instance banner
- Application version badge in the site header (`version: {version}` from `VERSION`)
- Hybrid PQ E2EE for chat keys and messages (IndexedDB wrapping keys, per-account identity)
- Disappearing messages: each message shows absolute disappear date/time (chat TTL default 60 days)
- Playwright E2E suite (`client/e2e`, `npm run test:e2e`) driving two independently signed-in real browser profiles through the real OAuth flow, real Postgres, and real client-side E2EE crypto, backed by a standalone mock OIDC server (`mockoidc`)
- Apple-like sign-in E2E spec (`client/e2e/apple-sign-in.spec.ts`) exercising `response_mode=form_post`, the one-time name payload, and `private_key_jwt` client authentication against a second mock issuer mode in `mockoidc`

#### Tooling & deployment

- Docker Compose for local development (PostgreSQL with `pg_cron`, server, client)
- AWS deployment tooling (ECR image build/push, EC2 compose, ALB setup scripts)
- Repo-root `VERSION` file shared by server builds, client builds, and documentation
- Cryptography NIST expert Cursor skill
- Cursor rule requiring `CHANGELOG.md` updates for notable changes
- Terraform (Phase 1 of #27): `terraform/modules/{ecr,github-oidc}` and `terraform/envs/shared`, codifying the account-wide ECR repositories and GitHub Actions OIDC role/policy already created manually; plan-only `terraform.yml` CI workflow (`fmt`, `validate`, gated `plan`)
- Terraform (Phase 2 of #27): `terraform/modules/{network,alb,ec2}` and `terraform/envs/{dev,prod}`, codifying per-environment networking (default-VPC data source), ALB (matching `create-alb.example.sh`'s target groups/listener rules), and EC2 (instance + security group + IAM instance profile); `terraform.yml` CI matrixed over `shared`/`dev`/`prod`
- Terraform (Phase 3 of #27, final phase): `terraform/modules/rds` and its wiring in `terraform/envs/{dev,prod}` — DB instance, subnet group, security group, and a parameter group folding in #20's `pg_cron` settings (`shared_preload_libraries=pg_cron`, `cron.database_name`); built without AWS credentials to audit the live RDS instances, so every identifying variable is required with no default and the module is not yet imported

### Changed

#### Tooling & deployment

- Terraform ALB module manages additional HTTPS listener rules via `additional_https_rules` (live-dev SPA routes at priorities 10–14: `/`, `/assets/*`, `/oauth/*`, `/chats`, `/instance.json`)
- Terraform `envs/dev` ALB defaults match live target groups (`dev-client` / health check `/` + `traffic-port`) and skip creating the reserved `"default"` DB subnet group
- Terraform AWS provider bumped to `>= 6.34.0, < 7.0.0` so `aws_lb_target_group_attachment` can be imported

### Fixed

- Client dev proxy now forwards `X-Forwarded-Host`/`X-Forwarded-Proto` on WebSocket upgrades too (not just plain HTTP requests), fixing live `chat.added`/`chat.unread` push under `npm run dev` against a local server
- Bearer ID-token verification now disambiguates OIDC providers that share the same issuer link by the token's audience (`aud`), not issuer alone -- an issuer-only lookup silently resolved to whichever provider name sorted first, authenticating a token against the wrong provider's `client_id` and failing every subsequent request with a 401 right after an otherwise-successful sign-in (surfaced by the e2e suite's "OIDC" and "GoogleE2E" mock providers, which intentionally point at the same mock issuer)

[NEXT RELEASE]: https://github.com/HrrrXppp/e2eetext/compare/main...HEAD
