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
- OIDC authentication with request-derived redirect URLs (Google seeded by default)
- User registration, search, and profile API
- Chats, messages, unread tracking, and a WebSocket event hub
- WebSocket endpoint and application route wiring
- `GET /health` returns `status` and `version`
- E2EE identity-key and chat key-wrap APIs; chat create requires per-member wraps
- Real integration test suite (`server/internal/integration`, `go test -tags=integration`) exercising real Postgres, real HTTP handlers, and real OIDC auth verification against a mock identity provider

#### Client

- React 19 + TypeScript + Vite SPA
- OIDC sign-in, session management, and landing page
- Chats UI, messaging, and real-time WebSocket updates
- Deploy-time dev instance banner
- Application version badge in the site header (`version: {version}` from `VERSION`)
- Hybrid PQ E2EE for chat keys and messages (IndexedDB wrapping keys, per-account identity)
- Playwright E2E suite (`client/e2e`, `npm run test:e2e`) driving two independently signed-in real browser profiles through the real OAuth flow, real Postgres, and real client-side E2EE crypto, backed by a standalone mock OIDC server (`mockoidc`)

#### Tooling & deployment

- Docker Compose for local development (PostgreSQL, server, client)
- AWS deployment tooling (ECR image build/push, EC2 compose, ALB setup scripts)
- Repo-root `VERSION` file shared by server builds, client builds, and documentation
- Cryptography NIST expert Cursor skill
- Cursor rule requiring `CHANGELOG.md` updates for notable changes

### Fixed

- Client dev proxy now forwards `X-Forwarded-Host`/`X-Forwarded-Proto` on WebSocket upgrades too (not just plain HTTP requests), fixing live `chat.added`/`chat.unread` push under `npm run dev` against a local server

[NEXT RELEASE]: https://github.com/HrrrXppp/e2eetext/compare/main...HEAD
