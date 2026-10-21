# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Release version is defined in `[VERSION](./VERSION)`.

On branches that are not yet merged to `main`, use `NEXT RELEASE` as the changelog heading. When a release is cut, add a dated semver section from `VERSION` under `NEXT RELEASE` and leave `NEXT RELEASE` in place for later changes.

## [NEXT RELEASE]

### Fixed

- Opening a chat now scrolls the message list to the latest message when there is no unread divider to jump to (no unread messages, or the first message is unread). It scrolls once per opened chat, so later updates do not move the list (#120)

## [0.3.9](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-18

### Added

- Client `/roadmap` page listing four planned milestones (base improvement, separated services with a custom load balancer, audio calls, video calls) with quarters and status badges, a Roadmap footer link, and a "Roadmap published" news item (#116)
- The landing page news list keeps a fixed maximum height and scrolls (keyboard focusable) (#116)

#### Tooling & deployment

- ALB priority 15 rule (dev and prod) and `sitemap.xml` include `/roadmap` (#116)

### Fixed

#### Tooling & deployment

- The prod RDS security group no longer drops the metrics Lambda's port 5432 rule. App and CloudShell access stay as standalone rules, and the security group does not reconcile inline ingress.
- The metrics Lambda reads the database secret on every run and trims it. A warm execution environment was keeping the first password, and a trailing newline made `sslmode` invalid.

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.9`

## [0.3.8](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-18

### Added

- AdSense readiness: `/privacy` Privacy Policy and `/terms` Terms of Service pages (draft content, owner should review legally), site footer with Privacy/Terms/Contact links and copyright, placeholder `ads.txt`, `sitemap.xml`, and a `Sitemap:` line in `robots.txt`; Terraform ALB rules (priorities 15-16, dev and prod) forward `/privacy`, `/terms`, `/robots.txt`, `/ads.txt` and `/sitemap.xml` to the client target group (#33)

### Fixed

#### Tooling & deployment

- Prod reuses dig's `monitoring` and `secretsmanager` interface endpoints instead of creating a second private-DNS pair in the same VPC. It opens TCP 443 on `dev-ec2-metrics-endpoints` for the prod metrics Lambda and the prod EC2 security group. Apply dig before prod so that security group stops using an inline rule, which would delete the prod rules.
- Prod RDS keeps the CloudShell self-ingress on port 5432 (`rds_ingress_allow_self`, default true).
- Dig adopts the existing TCP 443 rules on `dev-ec2-metrics-endpoints` instead of creating them again. A fresh authorize was rejected as `InvalidPermission.Duplicate`.

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.8`

## [0.3.7](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-17

### Fixed

#### Tooling & deployment

- The metrics `monitoring` and `secretsmanager` interface VPC endpoints allow TCP 443 from the EC2 instance security groups, not only from the metrics Lambda. Private DNS was sending the server's `PutMetricData` to an endpoint that dropped it, so `WebSocketConnections` publishes timed out.
- The read-only Terraform plan role can refresh the metrics Lambda, its log group, and its EventBridge rule (`iam:GetRole` on `*-metrics`, `logs:DescribeLogGroups`, `events:DescribeRule`). Apply `envs/shared` to update the live role.

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.7`

## [0.3.6](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-17

### Fixed

#### Tooling & deployment

- Dev and prod skip the metrics Lambda when `metrics_database_url_secret_arn` is empty, instead of planning an IAM policy with `Resource = ""`

### Added

- CloudWatch metrics (#37): a scheduled AWS Lambda (`lambdas/metrics`) publishes `Messages`, `Chats` and `Users` counts every minute, and each server node publishes its own `WebSocketConnections` (dimension `NodeId`) when `AWS_REGION` and `METRICS_CLOUDWATCH_NAMESPACE` are set. No public endpoint is added.

### Changed

#### Tooling & deployment

- New `terraform/modules/metrics` (Lambda, IAM, security group, EventBridge schedule, log group, optional VPC endpoints), wired into `terraform/envs/{dev,prod}` behind `metrics_enabled` (default `true`; requires `metrics_database_url_secret_arn`); the EC2 module role can grant `cloudwatch:PutMetricData` scoped to the namespace
- CI builds and tests the metrics Lambda and packages `metrics.zip`; the Terraform plan job builds it first
- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.6`

## [0.3.5](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-17

### Fixed

- Opening a chat no longer requests key wraps once per message; concurrent key lookups for a chat share one in-flight request (#102)

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.5`

## [0.3.4](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-14

### Added

- Landing page News: "E2EE Text is live in production" entry (October 11, 2026), listed above the project-start entry (#89)

#### Tooling & deployment

- `build-images.yml` now runs automatically when a push to `main` changes `VERSION`: it checks `client/package.json` matches `VERSION` (deployed `ec2_image_tag.tf` pins are independent of new builds), skips if the `vX.Y.Z` images already exist in ECR (manual dispatch has a `force` override), and queues builds so quick bumps run in order
- GitHub Actions ECR role policy gains `ecr:DescribeImages` on the two repos (apply `envs/shared` to update the live role)

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.4`

## [0.3.3](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-11

### Changed

- Footer keeps the "sustained through advertisement and donations" text, with "donations" linking to the Stripe donation page (opens in a new tab, `rel="noopener noreferrer"`); the URL lives in `client/src/lib/config.ts` and can be overridden at build time with `VITE_DONATE_URL` (#92)

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.3`

### Fixed

#### Client

- The donations link in the footer sits in the sentence with a normal space before it

## [0.3.2](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-11

### Added

#### Tooling & deployment

- Prod HTTPS uses a Terraform-managed ACM certificate in the ALB region (`e2eetext.com`, `.org`, `.net`, `.io`) when `acm_certificate_arn` is empty. Apply waits for DNS validation; this account has no Route 53 zones, so the CNAME records from output `acm_dns_validation` are created at the DNS host
- EC2 boot deploy installs Docker, git, and the AWS CLI when they are missing, and clones `https://github.com/HrrrXppp/e2eetext.git` (branch `main`) when the checkout is missing. `maintenance/ec2/.env` and the JSON config files are still supplied separately

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.2`

### Fixed

#### Client

- On a phone, the chat list is narrower and the message composer keeps the Send button inside the screen

#### Tooling & deployment

- Prod example RDS engine version is `16.15`. PostgreSQL `16.4` is no longer offered for new instances in us-east-2

## [0.3.1](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-11

### Fixed

#### Client

- Post-login private-key prompt keeps space between "Restore from a backup instead" and "Skip for now"

### Changed

#### Tooling & deployment

- Dig and prod EC2 boot-deploy `IMAGE_TAG` pins are `v0.3.1`

## [0.3.0](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD) - 2026-10-11

### Added

#### Server

- API rate limiting (#22): in-memory token-bucket middleware per route class (auth, search, message, default, WebSocket upgrade), keyed by client IP before authentication and by user afterward; `429` responses include `Retry-After`. Configurable via `rate_limits` in the server JSON config, with built-in defaults for local and production (see README)

#### Client

- Shared `Footer` component (site chrome parity with `SiteHeader`) now also renders on the chats page; footer styling moved out of the landing-only `landing__footer` BEM block into a page-agnostic `site-foot` block, with a compact non-scrolling variant for the height-locked chats app shell and the original spacing preserved on the landing page
- Export/import UI for the E2EE private identity key (#52): passphrase-protected `scrypt` + `AES-256-GCM` backup file, download-only (never clipboard), no minimum passphrase length; `Back up key` / `Restore key` buttons in the signed-in header, plus a one-time prompt right after a brand-new local keypair is generated offering to back up or restore instead. Import detects and rejects the legacy plaintext export format, and requires explicit confirmation before overwriting a different key already on file for the account.
- Desktop browser notifications for new messages when the tab is in the background (#48)

#### Tooling & deployment

- GitHub Actions ECR IAM: manage the live *inline* policy `github-actions-ecr-rolePolicy` via Terraform (not a managed policy attachment)
- Terraform `modules/terraform-plan-role` + `envs/shared` wiring: import/manage the CI `terraform-plan-role` inline policy (includes dig/prod EC2 instance-profile and SSM boot-deploy reads for `*-boot-deploy` documents)
- EC2 boot deploy managed by Terraform for dig and prod: systemd unit (`maintenance/ec2/e2eetext.service`) runs `deploy.sh` on every boot (ECR login, pull if needed, compose up); `IMAGE_TAG` is git-pinned per env in `envs/{dev,prod}/ec2_image_tag.tf` (dig and prod may differ)
- EC2 boot-deploy SSM install script is bash/`[`-safe (SSM defaults to `/bin/sh`), and dig's default checkout path is `/home/ubuntu/e2eetext`
- `deploy.sh` passes `/etc/e2eetext/deploy.env` to `docker compose` (and clears a stale shell `IMAGE_TAG`) so the Terraform pin wins over `maintenance/ec2/.env`; SSM install also rewrites `IMAGE_TAG` in that `.env` for older checkouts
- Terraform dig EC2 `name_prefix` defaults to `dev-ec2` (keeps the live Name tag)
- Terraform RDS pg_cron prep: require a custom parameter group when `enable_pg_cron` is set (not AWS `default.postgres*`) so `shared_preload_libraries` / `cron.database_name` (app DB) can be set; reboot the instance out-of-band after apply before migration `000006`

### Fixed

#### Client

- Chat list and message panel stay side by side at every width (narrower sidebar instead of stacking), so shrinking the window no longer jumps the chats list above the thread (#58)
- Chat list items stay packed at the top of the sidebar instead of stretching to fill its height (#50)
- Long unbroken message text wraps inside the bubble, and long author names truncate with an ellipsis (#63)
- Search users dialog stays within the phone screen width (#66)

## [0.2.0](https://github.com/HrrrXppp/e2eetext/compare/main...HEAD)

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
- Chat admins (#11): `chat_admins` table (migration `000008`) recording who's an admin, populated with the chat's creator at creation time; `Chat.isAdmin`/`Chat.createdBy` expose it via the existing chat APIs. Only the creator is an admin for now — a members-listing endpoint and promote/demote admin management are deferred to future changes. `chat_admins.user_id` is `ON DELETE RESTRICT`, so a user account can't be deleted while still an admin of a chat (#36); reassigning/demoting first is deferred to a future change



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
- Containerized Claude ticket-processing runner (#40): `maintenance/claude-runner/` (`Dockerfile`, `entrypoint.sh`, `docker-compose.yml`, `.env.example`) packages the recurring GitHub ticket-processing cycle from `CLAUDE.md` into an unattended Docker image that loops `claude -p` on a timer; a curated, secret-free `.claude/settings.json` permission allowlist is now committed (promoted from local-only) so the loop runs with `--permission-mode dontAsk` by default instead of blocking on interactive prompts, with `CLAUDE_PERMISSION_MODE=bypassPermissions` available as an explicit opt-in escape hatch



### Changed



#### Tooling & deployment

- Terraform ALB module manages additional HTTPS listener rules via `additional_https_rules` (live-dev SPA routes at priorities 10–14: `/`, `/assets/*`, `/oauth/*`, `/chats`, `/instance.json`)
- Terraform `envs/dev` ALB defaults match live target groups (`dev-client` / health check `/` + `traffic-port`) and skip creating the reserved `"default"` DB subnet group
- Terraform `envs/prod` ALB defaults now match `envs/dev` / live-dev shape (`fixed-404`, SPA rules 10–14, api priority 100, no `/health` listener rule, idle timeout 60, TG health thresholds 5/2, client health path `/`, server health port `8081`) instead of the script-shaped greenfield defaults
- Terraform AWS provider bumped to `>= 6.34.0, < 7.0.0` so `aws_lb_target_group_attachment` can be imported
- Terraform RDS `db_name` is nullable (default null): brownfield instances created with no initial `DBName` no longer ForceNew-replace when config defaulted to `"messenger"`
- Terraform RDS `app_database_name` / `rds_app_database_name` records the PostgreSQL DB used in `DATABASE_URL` without touching ForceNew CreateDBInstance `DBName` (live-dev: `dev_e2eetext`)



### Fixed



#### Client

- Client dev proxy now forwards `X-Forwarded-Host`/`X-Forwarded-Proto` on WebSocket upgrades too (not just plain HTTP requests), fixing live `chat.added`/`chat.unread` push under `npm run dev` against a local server



#### Server

- Bearer ID-token verification now disambiguates OIDC providers that share the same issuer link by the token's audience (`aud`), not issuer alone -- an issuer-only lookup silently resolved to whichever provider name sorted first, authenticating a token against the wrong provider's `client_id` and failing every subsequent request with a 401 right after an otherwise-successful sign-in (surfaced by the e2e suite's "OIDC" and "GoogleE2E" mock providers, which intentionally point at the same mock issuer)

