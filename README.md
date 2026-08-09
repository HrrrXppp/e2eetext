# Messenger

Monorepo messenger (pre-MVP) with a Go HTTP API and a React frontend. **New chats and messages use end-to-end encryption (E2EE v1)** — the server stores wrapped chat keys and opaque ciphertext only. Authentication uses OIDC providers stored in PostgreSQL (Google and Apple are seeded by default).

## Stack

| Layer    | Technology                          |
|----------|-------------------------------------|
| Backend  | Go 1.25, stdlib `net/http`, pgx     |
| Frontend | React 19, TypeScript, Vite 8        |
| Database | PostgreSQL 16                       |
| Auth     | OIDC (Google, Apple), stateless ID tokens |

## Project structure

```
messenger/
├── VERSION          # release version (server + client)
├── CHANGELOG.md     # release notes
├── client/          # React SPA
├── server/          # Go API
│   ├── cmd/messenger/
│   └── internal/
│       ├── app/           # route wiring
│       ├── config/        # env configuration
│       ├── database/      # postgres + migrations
│       ├── domain/        # entities
│       ├── handler/       # HTTP handlers
│       ├── middleware/    # logging, auth
│       ├── repository/    # data access
│       └── service/       # business logic
└── docker-compose.yml     # local PostgreSQL
├── maintenance/
│   ├── ecr/               # build + push images to AWS ECR
│   ├── alb/               # ALB setup example (AWS CLI)
│   ├── ec2/               # single-EC2 production compose
│   └── claude-runner/     # containerized Claude ticket-processing runner
└── terraform/
    ├── modules/           # reusable ecr, github-oidc (more in later phases)
    └── envs/
        └── shared/        # account-wide root config (Phase 1: ECR + OIDC)
```


## Prerequisites

- [Go](https://go.dev/) 1.25+
- [Node.js](https://nodejs.org/) 20.19+ or 22.12+ (see `.nvmrc`)

Install the required Node version with [nvm](https://github.com/nvm-sh/nvm):

```bash
nvm install
nvm use
node -v
```

From the repo root, `nvm install` reads `.nvmrc` (Node 22).
- [Docker](https://www.docker.com/) (for PostgreSQL)
- Google Cloud OAuth credentials (for sign-in)

## Quick start

### 1. Start PostgreSQL

The server needs PostgreSQL running before it starts. From the repo root:

```bash
docker compose up -d postgres
```

The server waits up to 60 seconds for PostgreSQL on startup. To reset the database (e.g. after migration changes):

```bash
docker compose down -v && docker compose up -d postgres
```

### 2. Configure the server

Copy env and config files:

```bash
cp .env.example .env
cp server/config.example.json server/config.json
```

Edit `.env` — set `DATABASE_URL` (and keep `CONFIG_PATH=server/config.json` from the example). Edit `server/config.json` and set OAuth credentials per provider slug:

```json
{
  "oauth_credentials": {
    "google": {
      "client_id": "your-client-id",
      "client_secret": "your-client-secret"
    },
    "apple": {
      "client_id": "your-services-id",
      "private_key_jwt_issuer": "your-team-id",
      "private_key_jwt_key_id": "your-key-id",
      "private_key_jwt_algorithm": "ES256",
      "private_key_jwt_private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
    }
  }
}
```

Get Google credentials from [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials. Create an **OAuth 2.0 Client ID** (Web application).

Each provider's `oidc_providers` DB row (see [Database](#database) below) says which of these two shapes it expects, via `client_secret_strategy`:

- **`static`** (Google) — reads `client_secret` as a plain configured secret.
- **`private_key_jwt`** (Apple, RFC 7523 / OpenID Connect Core §9) — Apple has no static client secret at all; the server signs a short-lived ES256 JWT per request instead, using the PEM-encoded EC private key you generate for your "Sign in with Apple" key. `client_id` is the Services ID; `private_key_jwt_issuer` is your Apple Developer Team ID; `private_key_jwt_key_id` is the Key ID shown when you created the key; `private_key_jwt_algorithm` defaults to `ES256` if omitted. `private_key_jwt_issuer` and `private_key_jwt_key_id` are required whenever `private_key_jwt_private_key` is set, and `private_key_jwt_algorithm` (if given explicitly) must be `ES256` — the server validates this at config load and refuses to start otherwise, rather than minting a JWT the IdP would only reject later at token exchange.

This is a generic OAuth2 client-authentication mechanism (not Apple-specific) — any provider row can opt into it by setting `client_secret_strategy = 'private_key_jwt'`.

**Authorized redirect URIs** — register every origin where the app runs (the server builds the callback URL from the request host):

- Local dev: `http://localhost:5173/api/v1/auth/google/callback` (and `/api/v1/auth/apple/callback`)
- Production: `https://app.example.com/api/v1/auth/google/callback` (and `/api/v1/auth/apple/callback`)

OAuth callbacks are derived from `X-Forwarded-Host` and `X-Forwarded-Proto` on each request (the Vite dev proxy sets these in local dev). The callback route accepts both `GET` (the default 302-redirect flow, e.g. Google) and `POST` (required by providers using `response_mode=form_post`, e.g. Apple whenever name/email scopes are requested) at the same URL.

The server loads `.env` for runtime settings (`DATABASE_URL`, `CONFIG_PATH`, etc.) and the JSON file at `CONFIG_PATH` for OAuth secrets.

### 3. Start the server

Make sure PostgreSQL is running first (see step 1).

```bash
cd server
go run ./cmd/messenger
```

The server waits up to 60 seconds for PostgreSQL, runs migrations, then listens on `http://localhost:8080`.

### 4. Start the client

In a second terminal:

```bash
cd client
npm install
npm run dev
```

Open **http://localhost:5173**.

## Environment variables

| Variable        | Default                                                              | Description                        |
|-----------------|----------------------------------------------------------------------|------------------------------------|
| `SERVER_ADDR`   | `:8080`                                                              | HTTP listen address                |
| `DATABASE_URL`  | _(required)_                                                         | PostgreSQL connection string       |
| `CONFIG_PATH`   | _(required)_                                                         | Path to server JSON config file (e.g. `server/config.json`) |

OAuth client credentials live in `server/config.json` under `oauth_credentials`, keyed by provider slug (e.g. `google`).

## Versioning

The release version lives in the repo-root `VERSION` file (currently semver, e.g. `0.1.0`). Bump it when cutting a release:

- **Server** — `GET /health` returns `version`; Docker builds stamp it via `-ldflags`.
- **Client** — Vite injects `VITE_APP_VERSION` from `VERSION`; the header shows `version: {version}`.

Keep `client/package.json` `version` in sync with `VERSION` for npm metadata. Record user-facing release notes in [`CHANGELOG.md`](./CHANGELOG.md). On feature branches, use a **`NEXT RELEASE`** section until the change is merged to `main` and versioned.

## API

All JSON and WebSocket endpoints are versioned under `/api/v1`. `/health` is unversioned.

### Public

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (`status`, `version`) |
| `GET` | `/api/v1/auth/providers` | List OIDC providers |
| `GET` | `/api/v1/auth/{provider}/login` | Start OIDC login (e.g. `google`, `apple`) |
| `GET`, `POST` | `/api/v1/auth/{provider}/callback` | OAuth callback; redirects to client with tokens. `POST` is for providers using `response_mode=form_post` (Apple) |
| `POST` | `/api/v1/auth/refresh` | Refresh tokens (`provider`, `refreshToken` in JSON body) |

### Protected

All routes below require `Authorization: Bearer <id_token>` (or `?token=` for the WebSocket).

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/search?q={query}` | Search users by name or scoped ID (min. 3 characters) |
| `GET` | `/api/v1/user?subject=&oidc_provider_id=` | List users matching filter |
| `POST` | `/api/v1/user` | Create user from auth token (optional body: `{ "skip_profile": true }`) |
| `GET` | `/api/v1/user/{nodeId}/{localId}` | Get user by scoped ID |
| `PATCH` | `/api/v1/user/{nodeId}/{localId}` | Update display name (`name` in JSON body) |
| `PUT` | `/api/v1/user/{nodeId}/{localId}/identity-key` | Upload E2EE identity public key (own user only) |
| `GET` | `/api/v1/user/{nodeId}/{localId}/identity-key` | Fetch a user's identity public key (authenticated) |
| `GET` | `/api/v1/chat?user_id=` | List chats for a user (scoped `user_id`; response `id` is scoped) |
| `POST` | `/api/v1/chat` | Create E2EE chat (`name`, `users_uids`, optional `disappear_after_minutes`, `e2ee.key_id`, `e2ee.wraps`) |
| `GET` | `/api/v1/chat/{nodeId}/{localId}/key-wraps` | List chat-key wraps for the current member |
| `GET` | `/api/v1/message?chat_id=` | List messages in a chat (scoped `chat_id`; response `chatId` is scoped) |
| `POST` | `/api/v1/message` | Create message (`chat_id`, `user_id`, `data`; scoped IDs) |
| `PATCH` | `/api/v1/message/{nodeId}/{localId}` | Mark message read (`{"unread": false}`) |
| `GET` | `/api/v1/ws` | WebSocket for realtime events (`chat.added`, `chat.unread`) |

Scoped resource IDs use the form `{nodeId}/{localId}` (e.g. `99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111`).

## End-to-end encryption (v1)

Hybrid **ML-KEM-768 + X25519** identity keys wrap a per-chat symmetric key `K_chat`; message bodies are **AES-256-GCM** (max **100 KiB** plaintext). The server stores public identity keys, wrapped chat keys, and opaque ciphertext in `messages.data` — it cannot decrypt content.

### Algorithm suite

| Purpose | Algorithm |
|---------|-----------|
| Identity KEM | ML-KEM-768 + X25519 hybrid (`ml_kem768_x25519`) |
| Wrap `K_chat` | Hybrid encaps → HKDF-SHA-256 → AES-256-GCM |
| Message body | HKDF-SHA-256 from `K_chat` → AES-256-GCM |

### Wire types (binary fields are base64url)

**IdentityPublicKey** (server): `{ "v": 1, "alg": "hybrid-kem-mlkem768-x25519", "publicKey": "…" }`

**ChatKeyWrap** (per member): `{ "v": 1, "alg": "hybrid-kem-mlkem768-x25519-aes256gcm", "keyId", "kemCiphertext", "nonce", "ciphertext" }`

**MessageEnvelope** (`messages.data`): `{ "v": 1, "alg": "aes256gcm-chat-key", "keyId", "nonce", "ciphertext" }`

### Client flow

1. **Sign-in** — generate identity keypair if missing; `PUT identity-key`.
2. **Create chat** — generate `K_chat` + `keyId`; fetch each member's public key; build wraps; `POST /chat` with `e2ee`.
3. **Open chat** — `GET key-wraps`; unwrap `K_chat`; cache locally (wrapped with the non-extractable device key).
4. **Send / receive** — encrypt/decrypt `MessageEnvelope` in the client.

Private identity ciphertext is stored per account in `localStorage` (`e2ee_identity_v1:{userId}`) and encrypted with a **non-extractable AES-256-GCM** wrapping key kept in IndexedDB (Web Crypto). That blocks XSS from reading raw wrapping-key bytes out of storage. Active same-origin XSS can still invoke decrypt APIs while the tab is unlocked — a passphrase lock or isolated crypto origin would be needed to close that remaining gap. Export/import backup JSON is available via `exportStoredIdentityBackup(userId)` / `importStoredIdentityBackup(userId, …)`. Sign-out clears that account's local E2EE material.

### v1 limits

- All chats require E2EE (`e2ee` payload mandatory on `POST /chat`).
- One device per user (`device_id = default`).
- No key rotation on member remove yet.
- Chat names remain plaintext.
- Legacy plaintext messages (non-envelope `data`) are shown as-is.

## Disappearing messages

Chat-level TTL only. Column `chats.disappear_after_minutes` (API `disappearAfterMinutes`), **default 60 days (86400 minutes)**. A message is live while:

`messages.created_at + disappear_after_minutes > now()`

There is no per-message `expires_at`. Changing a chat’s TTL retargets existing messages.

- **List / unread** — unchanged queries; expired rows are removed by purge.
- **Purge** — migration `000006_disappear` creates `purge_expired_messages()` and schedules it with **`pg_cron`** every minute. Local Compose builds Postgres inline (`dockerfile_inline` in `docker-compose.yml`: Postgres 16 + `postgresql-16-cron`) with `shared_preload_libraries=pg_cron` and `cron.database_name=messenger`. On AWS RDS, enable the `pg_cron` extension in the parameter group / allowed extensions before migrating.
- **Client** — default 60-day TTL on create; countdown from `createdAt` + chat TTL; local drop on a timer.

## Authentication flow

```
Browser → GET /api/v1/auth/google/login
       → Google consent screen
       → GET /api/v1/auth/google/callback?code=…&state=…
       → Server verifies token
       → Redirect to /oauth/callback#id_token=…&provider=google
       → Client stores ID token in localStorage
       → Client GET /api/v1/user?subject=…&oidc_provider_id=…; POST /api/v1/user if missing (identity from token)
       → Subsequent API calls send Authorization: Bearer <id_token>
```

The server verifies the ID token against the provider issuer on each protected request and reads user claims (`sub`, `email`, `name`) from the token — no server-side sessions.

Apple follows the same flow with two differences, both driven by its `oidc_providers` row rather than special-cased Go code: the consent screen `POST`s back to the callback URL (`response_mode=form_post`) instead of a 302 redirect, and — only on the very first authorization for a given Apple user — that `POST` carries a one-time `user` JSON field (name only; the server requests just the `name` scope, not `email` — it doesn't keep the user's email address); Apple's ID token itself never carries a name claim at all. The server parses that one-time field and forwards the name to the client via the callback redirect (`&name=…`), which relays it on the next `POST /api/v1/user` call; the server only uses it if the verified ID token didn't already carry a name.

## Database

Schema is defined in a single migration:

```
server/internal/database/migrations/000001_init.up.sql
```

Tables:

- **oidc_providers** — `id`, `name`, `link`, `picture` (BYTEA), `scopes` (space-separated, e.g. `openid profile`), `response_mode` (nullable, e.g. `form_post`), `client_secret_strategy` (`static` or `private_key_jwt`)
- **users** — linked to an OIDC provider via `oidc_provider_id` + `subject`

Google and Apple are inserted as the default providers (migrations `000001_init` and `000007_apple_oidc`). Provider-specific OAuth peculiarities (scopes, response mode, client authentication) are data on the row, not hardcoded per-provider branches in Go — a new provider is a migration + `server/config.json` credentials, not a code change.

## Development

### Build

```bash
# Server
cd server && go build ./...

# Client
cd client && npm run build
```

### Test

```bash
# Server
cd server && go test ./...

# Client
cd client && npm test
```

### Integration tests (server, real Postgres)

`server/internal/integration` exercises the real HTTP handlers, real Postgres
repositories, and real auth verification — no mocks — against a mock OIDC
issuer that mints real signed ID tokens, so the exact same middleware chain
production traffic hits gets exercised. It's build-tag gated (`integration`)
so it's never compiled or run by the default `go test ./...`.

```bash
docker run -d --name e2eetext-integration-db \
  -e POSTGRES_USER=messenger -e POSTGRES_PASSWORD=messenger -e POSTGRES_DB=messenger \
  -p 55432:5432 postgres:16-alpine

cd server
INTEGRATION_DATABASE_URL="postgres://messenger:messenger@127.0.0.1:55432/messenger?sslmode=disable" \
  go test -tags=integration ./internal/integration/...
```

Migrations run automatically on first connect. Tests skip (not fail) if
`INTEGRATION_DATABASE_URL` (or `DATABASE_URL`) isn't set.

### End-to-end tests (client, real browser)

`client/e2e` drives a real Chromium browser through two independently
signed-in profiles — real OAuth authorization-code flow (against a
throwaway mock identity provider, `mockoidc`), a real Go server,
real Postgres, and real client-side ML-KEM/AES-GCM crypto — creating a chat
and exchanging a message to prove E2EE actually round-trips between two
separate browser identities, not just that the UI renders.

```bash
docker run -d --name e2eetext-e2e-db \
  -e POSTGRES_USER=messenger -e POSTGRES_PASSWORD=messenger -e POSTGRES_DB=messenger \
  -p 55433:5432 postgres:16-alpine

cd client
npx playwright install chromium   # first run only
E2E_DATABASE_URL="postgres://messenger:messenger@127.0.0.1:55433/messenger?sslmode=disable" \
  npm run test:e2e
```

`client/e2e/global-setup.ts` boots the mock identity provider, the Go
server, and `vite dev` itself — no separate `npm run dev` needed. Point it
at a disposable database; each run seeds an "OIDC" sign-in provider and
creates fresh random test users, so it's safe to reuse the same one across
runs. This suite is comprehensive rather than fast (real network round
trips, real crypto, two full sign-ins) — on a loaded machine, prefer the
Go integration suite above for quick iteration and treat this one as a
pre-release sanity check.

`global-setup.ts` also seeds a "GoogleE2E" provider pointing at the same
Google-shaped default mode as "OIDC" (GET-redirect authorize, static client
secret, ID token carries a `name` claim directly), kept as its own
name/slug so `e2e/google-sign-in.spec.ts` — which exercises a person
authorizing via that provider end-to-end, including that the ID token's
`name` claim registers as their initial display name — doesn't share
provider state with `golden-path.spec.ts`'s use of "OIDC".

`mockoidc` also serves an "Apple-like" mode under `/apple` (form_post
response, one-time name/email payload, `private_key_jwt` client
authentication) — `global-setup.ts` seeds a second "AppleE2E" provider
pointing at it and feeds the server a matching EC keypair, so
`e2e/apple-sign-in.spec.ts` exercises the real `private_key_jwt` code path
against the mock rather than a bypass. It leaves the default "OIDC"
provider and `golden-path.spec.ts` untouched.

### Client dev proxy

Vite proxies `/api` and `/health` to `http://localhost:8080`, so the browser uses a single origin (`localhost:5173`) during development.

### Ports

| Service    | Port  |
|------------|-------|
| Frontend   | 5173  |
| Backend    | 8080  |
| PostgreSQL | 5432  |

If port 5432 is already in use, change the mapping in `docker-compose.yml` and update `DATABASE_URL` accordingly.

## Production on AWS (single EC2 + ALB)

Run the app on **one EC2 instance** with **PostgreSQL outside the instance** (recommended: [Amazon RDS](https://aws.amazon.com/rds/postgresql/)). An **Application Load Balancer** terminates HTTPS and routes API traffic to the server and everything else to the client SPA.

```
Internet → ALB (:443, ACM certificate)
              ├─ /api*, /health → EC2:8081 → server container
              └─ /, /chats, …   → EC2:8080 → client container
           RDS PostgreSQL (private subnet)
```

The server reads `X-Forwarded-Host` and `X-Forwarded-Proto` from the ALB for OAuth redirects.

### 1. Create RDS PostgreSQL

- Engine: PostgreSQL 16
- Database name: `messenger`
- Note the endpoint, username, and password
- Security group: allow **inbound TCP 5432** from the EC2 security group (or EC2 private IP)
- Keep RDS in the same VPC as EC2; do not expose PostgreSQL to the public internet

`terraform/modules/rds` ([Infrastructure as code (Terraform)](#infrastructure-as-code-terraform) below) can manage this instance's subnet group, security group, and `pg_cron` parameter group going forward, but is not yet imported against either environment — this manual walkthrough remains the documented path until that import is confirmed zero-diff. See [#20](https://github.com/HrrrXppp/e2eetext/issues/20) for the full `pg_cron`-on-RDS parameter-group/reboot procedure this step still requires today.

Connection string example:

```text
postgres://messenger:PASSWORD@your-db.region.rds.amazonaws.com:5432/messenger?sslmode=require
```

The server runs migrations automatically on startup.

### 2. Create ECR repositories

The documented path is now `terraform apply` — see [Infrastructure as code (Terraform)](#infrastructure-as-code-terraform) below, which also codifies the GitHub Actions OIDC role used by `build-images.yml`. The script below remains as a manual fallback/bootstrap option:

On your **local machine** (AWS CLI configured):

```bash
cp maintenance/ecr/.env.example maintenance/ecr/.env
# edit AWS_ACCOUNT_ID and AWS_REGION

bash maintenance/ecr/create-repos.sh
```

Creates `e2eetext-server` and `e2eetext-client` repositories in ECR.

### 3. Build images locally and push to ECR

On your **local machine**:

```bash
bash maintenance/ecr/build-and-push.sh
```

This builds `server` and `client` Docker images and pushes them to ECR. Set `IMAGE_TAG` in `maintenance/ecr/.env` to version releases (default: `latest`). On Apple Silicon Macs, `DOCKER_PLATFORM=linux/amd64` is set by default for EC2.

Requires IAM permissions: `ecr:GetAuthorizationToken`, `ecr:BatchCheckLayerAvailability`, `ecr:PutImage`, `ecr:InitiateLayerUpload`, `ecr:UploadLayerPart`, `ecr:CompleteLayerUpload`.

Use the **same client image** on prod and dev. The dev banner is controlled at **deploy time**, not build time:

- Mount `maintenance/ec2/config/instance.json` into the client container (see `docker-compose.yml`).
- **Production EC2:** `cp maintenance/ec2/config/instance.production.example.json maintenance/ec2/config/instance.json`
- **Development EC2:** `cp maintenance/ec2/config/instance.development.example.json maintenance/ec2/config/instance.json`

The client fetches `/instance.json` at runtime.

ALB cannot inject HTML into the SPA; this belongs in the client with deploy-time config (same idea as server `config.json`).

### 4. Create EC2

- AMI: Amazon Linux 2023 or Ubuntu
- Instance type: `t3.small` or larger
- IAM role: attach **AmazonEC2ContainerRegistryReadOnly** (so EC2 can pull from ECR)
- Security group inbound:
  - **22** — SSH (your IP only)
  - **8080** — client (from ALB security group only)
  - **8081** — server (from ALB security group only)

Do **not** expose 80/443 on the EC2 instance — the ALB handles public HTTP(S).

### 5. Create Application Load Balancer

1. Request an **ACM certificate** for your domain (e.g. `app.example.com`) in the same region as the ALB.
2. Create two **target groups** (type: instance, protocol: HTTP):
   - **server** — port **8081**, health check path `/health`
   - **client** — port **8080**, health check path `/health`
3. Register your EC2 instance in **both** target groups (same instance, different ports).
4. Create an **internet-facing ALB** in public subnets with an ALB security group allowing **80** and **443** from the internet.
5. Set ALB **idle timeout** to **3600** seconds (WebSocket support on `/ws`).
6. Add an **HTTPS (443)** listener with your ACM certificate. Default action: forward to the **client** target group.
7. Add **listener rules** (lower priority number = evaluated first):

| Priority | Path pattern | Target |
|----------|--------------|--------|
| 10 | `/api*` | server |
| 20 | `/health` (exact) | server |
| default | `/*` | client |

`/oauth/callback` is a client SPA route and is served by the default listener rule (not under `/api*`).

8. Add an **HTTP (80)** listener that redirects to HTTPS.

An example AWS CLI script with these steps is in `maintenance/alb/create-alb.example.sh`.

### 6. Deploy on EC2

```bash
git clone <your-repo-url> e2eetext
cd e2eetext

sudo bash maintenance/ec2/setup-docker.sh
# log out and back in so docker group applies

# AWS CLI on EC2 (for ECR login) — Amazon Linux:
sudo dnf install -y awscli
# Ubuntu: sudo apt install -y awscli

cp maintenance/ec2/.env.example maintenance/ec2/.env
cp maintenance/ec2/config/config.example.json maintenance/ec2/config/config.json
```

Edit `maintenance/ec2/.env`:

| Variable | Example |
|----------|---------|
| `DATABASE_URL` | RDS connection string above |
| `AWS_ACCOUNT_ID` | same as in `maintenance/ecr/.env` |
| `AWS_REGION` | e.g. `us-east-1` |
| `IMAGE_TAG` | `latest` (or the tag you pushed) |

Edit `maintenance/ec2/config/config.json` with Google OAuth `client_id` and `client_secret`.

Pull images from ECR and start:

```bash
bash maintenance/ec2/deploy.sh
```

To update after a new push:

```bash
# locally
bash maintenance/ecr/build-and-push.sh

# on EC2
IMAGE_TAG=latest bash maintenance/ec2/deploy.sh
```

Point DNS `app.example.com` to the ALB (Route 53 alias record or CNAME to the ALB DNS name).

### 7. Google OAuth

In [Google Cloud Console](https://console.cloud.google.com/) → Credentials, add **Authorized redirect URI**:

`https://app.example.com/api/v1/auth/google/callback`

(Use your real domain. Local dev uses `http://localhost:5173/api/v1/auth/google/callback`.)

### 8. Verify

```bash
curl https://app.example.com/health
```

Open `https://app.example.com` in the browser and sign in.

### Files

| Path | Purpose |
|------|---------|
| `maintenance/ecr/.env.example` | AWS account, region, image tag |
| `maintenance/ecr/create-repos.sh` | Create ECR repositories |
| `maintenance/ecr/build-and-push.sh` | Build locally and push to ECR |
| `maintenance/alb/create-alb.example.sh` | Example ALB + listener rules (AWS CLI) |
| `maintenance/ec2/docker-compose.yml` | Pull ECR images; expose 8080/8081 for ALB |
| `maintenance/ec2/deploy.sh` | ECR login, pull, and start on EC2 |
| `maintenance/ec2/.env.example` | `DATABASE_URL`, ECR settings |
| `maintenance/ec2/setup-docker.sh` | Install Docker on a fresh EC2 |

`maintenance/ec2/.env` and `maintenance/ecr/.env` are gitignored — do not commit credentials.

### Troubleshooting deploy

**`not a directory` when mounting config:** Docker created a folder at `server/config.json` because the file did not exist on the first run. Fix:

```bash
docker compose --env-file maintenance/ec2/.env -f maintenance/ec2/docker-compose.yml down
rm -rf server/config.json
cp maintenance/ec2/config/config.example.json maintenance/ec2/config/config.json
# edit OAuth credentials in maintenance/ec2/config/config.json
bash maintenance/ec2/deploy.sh
```

**Server won't start:** `docker compose --env-file maintenance/ec2/.env -f maintenance/ec2/docker-compose.yml logs server`

| Log message | Fix |
|-------------|-----|
| `database unavailable` | `DATABASE_URL`, RDS security group port 5432 from EC2 |
| `load config` | `CONFIG_PATH` must point to a valid JSON file (e.g. `maintenance/ec2/config/config.json` on EC2) |
| `exec format error` | Rebuild with `DOCKER_PLATFORM=linux/amd64` |

## Infrastructure as code (Terraform)

AWS infrastructure is being migrated from the manual/scripted setup above to Terraform, in phases tracked by [#27](https://github.com/HrrrXppp/e2eetext/issues/27):

- **Phase 1 (done):** `terraform/envs/shared` — the two ECR repositories and the GitHub Actions OIDC role/policy, all account-wide with no dev/prod split.
- **Phase 2 (done):** `terraform/envs/{dev,prod}` — networking, ALB, and EC2, which are genuinely per-environment.
- **Phase 3 (module built, not yet imported):** `terraform/modules/rds` and `terraform/envs/{dev,prod}` — RDS PostgreSQL, its subnet group, security group, and a parameter group carrying the `pg_cron` settings from [#20](https://github.com/HrrrXppp/e2eetext/issues/20)'s plan (`shared_preload_libraries=pg_cron`, `cron.database_name`). Unlike Phases 1/2, this phase was implemented with **no AWS credentials available**, so none of the live RDS instance's real config (engine version, instance class, storage, subnet/parameter group names, ...) could be audited or verified. Every identifying RDS variable in `terraform/envs/{dev,prod}` is therefore required with no default — see [Phase 3 — RDS: not yet imported](#phase-3--rds-not-yet-imported) below for what's needed before the first `terraform import`/`apply`.

This repo is the last phase tracked by #27 — once Phase 3's import is confirmed zero-diff and merged, #27 closes.

```
terraform/
├── modules/
│   ├── ecr/            # the e2eetext-server / e2eetext-client repos
│   ├── github-oidc/    # OIDC provider + IAM role/policy for build-images.yml
│   ├── network/         # data-source lookup of the VPC/subnets to deploy into
│   ├── alb/              # ALB + target groups + listener rules (matches create-alb.example.sh)
│   ├── ec2/              # instance + security group + IAM instance profile
│   └── rds/              # DB instance + subnet group + security group + pg_cron parameter group
└── envs/
    ├── shared/          # root config instantiating ecr + github-oidc (Phase 1)
    ├── dev/              # root config instantiating network + alb + ec2 + rds for dev (Phase 2/3)
    └── prod/             # same, for prod (Phase 2/3)
```

`terraform/modules/network` is deliberately data-source-only (it never creates/modifies/destroys a VPC or subnets) — it looks up the account's default VPC and subnets unless overridden, since nothing in this repo's scripts or docs ever creates a custom VPC. `terraform/modules/alb` also does not create the ALB's own security group; like `create-alb.example.sh`, it takes an existing security group ID as input. The `alb` and `ec2` modules default to the topology `create-alb.example.sh` describes, but expose overrides (target-group names, idle timeout, listener default action, rule priority/patterns, pre-existing instance SGs, optional IAM instance profile) because the live, hand-made dev stack diverges from the script on all of those — `terraform/envs/dev`'s defaults record the audited live values. `terraform/modules/rds` has no such audited defaults for identifying attributes (engine version, instance class, storage, subnet/parameter group names, master username, ...) — every one is a required variable, the same "describe reality, don't guess" pattern `modules/ec2` uses for `ami_id`/`instance_type`, because no live RDS audit was possible while writing it.

### One-time backend bootstrap

State is stored in S3 with native S3 state locking (Terraform >= 1.10, no DynamoDB table needed). The bucket referenced in `terraform/envs/shared/versions.tf` (`e2eetext-terraform-state`) does not exist yet — create it once, with versioning enabled, before the first `terraform init`:

```bash
aws s3api create-bucket --bucket e2eetext-terraform-state --region us-east-2 \
  --create-bucket-configuration LocationConstraint=us-east-2
aws s3api put-bucket-versioning --bucket e2eetext-terraform-state \
  --versioning-configuration Status=Enabled

# State holds IAM/ECR topology (and later more sensitive env data) — keep
# the bucket private and encrypted at rest from the start:
aws s3api put-public-access-block --bucket e2eetext-terraform-state \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
aws s3api put-bucket-encryption --bucket e2eetext-terraform-state \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
```

### One-time import bootstrap (`envs/shared` — Phase 1)

The ECR repos and the GitHub OIDC provider/role/policy already exist — they were created by hand (`maintenance/ecr/create-repos.sh`, and a manual OIDC setup for `build-images.yml`). Terraform must **import** them, not create them, or `apply` will try to recreate live resources and fail/conflict. Before the first `terraform apply`:

```bash
cd terraform/envs/shared
cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars: set github_actions_role_name / github_actions_policy_name
# to whatever the manually-created IAM role/policy are actually named in AWS.

terraform init

# ECR repositories
terraform import 'module.ecr.aws_ecr_repository.this["e2eetext-server"]' e2eetext-server
terraform import 'module.ecr.aws_ecr_repository.this["e2eetext-client"]' e2eetext-client

# GitHub Actions OIDC identity provider
terraform import module.github_oidc.aws_iam_openid_connect_provider.github \
  arn:aws:iam::<AWS_ACCOUNT_ID>:oidc-provider/token.actions.githubusercontent.com

# IAM role assumed by build-images.yml (secrets.AWS_ROLE_ARN)
terraform import module.github_oidc.aws_iam_role.github_actions_ecr_push <ROLE_NAME>

# IAM policy granting ECR push, attached to the role above
terraform import module.github_oidc.aws_iam_policy.ecr_push \
  arn:aws:iam::<AWS_ACCOUNT_ID>:policy/<POLICY_NAME>

# Attachment of the policy above to the role above
terraform import module.github_oidc.aws_iam_role_policy_attachment.ecr_push \
  <ROLE_NAME>/arn:aws:iam::<AWS_ACCOUNT_ID>:policy/<POLICY_NAME>

# terraform.yml plan role (inline policy — not a managed policy attachment)
terraform import module.terraform_plan_role.aws_iam_role.this terraform-plan-role
terraform import module.terraform_plan_role.aws_iam_role_policy.this \
  terraform-plan-role:terraform-plan-rolePolicy
```

Replace `<AWS_ACCOUNT_ID>`, `<ROLE_NAME>`, and `<POLICY_NAME>` with the real values (`aws sts get-caller-identity`, `aws iam list-roles` / `list-policies` if the exact names aren't already known). After importing the Phase 1 ECR/OIDC resources, `terraform plan` should show **no changes** for those. Importing `terraform-plan-role` may plan an in-place **inline policy update** (expanded dig/prod EC2 + SSM reads) — review that diff, then `terraform apply` in `envs/shared` with admin credentials (not CI) to publish the policy.

### One-time import bootstrap (`envs/dev` — Phase 2)

A live dev EC2/ALB/VPC stack already exists (confirmed by the repo owner on [#27](https://github.com/HrrrXppp/e2eetext/issues/27)) — it was created by hand, and an audit against the AWS API (2026-07-26) showed it diverges from `maintenance/alb/create-alb.example.sh` in nearly every detail: the ALB is named `dev-e2eetext` (not `e2eetext-dev`), the target groups are named `dev-server` and `client`, the idle timeout is 60 (not the script's 3600), the HTTPS listener's default action is a fixed 404 (client routing lives in hand-made path rules at priorities 10–14 instead of a catch-all forward), the `/api/*` rule sits at priority 100, there is no `/health` listener rule, the instance runs with **no IAM instance profile** and with hand-made security groups (`launch-wizard-2` + `ec2-rds-1`), and nothing is tagged. `terraform/envs/dev`'s variable defaults record all of those audited values, so it describes what is actually live with **no `terraform.tfvars` needed** — `terraform plan` works out of the box (it looks the ALB security group and ACM certificate ARN up from the live ALB by name, so no account-specific IDs are committed to the repo).

As with Phase 1, Terraform must **import** the live resources, not create them:

```bash
cd terraform/envs/dev
terraform init

# Required for import only (not for a plain plan): the live instance's real
# subnet — module.network's subnet ordering isn't guaranteed, and a
# subnet_id mismatch on aws_instance forces a replace.
echo 'ec2_subnet_id = "<DEV_INSTANCE_SUBNET_ID>"' > terraform.tfvars

# VPC/subnets are data-sourced, not managed resources — nothing to import
# there. Same for the instance's SGs and the ALB's SG (pre-existing,
# looked up by name), and there is no IAM role/instance profile to import
# (the live instance runs without one; create_ec2_iam_instance_profile
# defaults to false accordingly).

# EC2 instance
terraform import module.ec2.aws_instance.this <DEV_INSTANCE_ID>

# ALB, its two target groups, and its listeners/rules
terraform import module.alb.aws_lb.this <DEV_ALB_ARN>
terraform import module.alb.aws_lb_target_group.server <DEV_SERVER_TG_ARN>
terraform import module.alb.aws_lb_target_group.client <DEV_CLIENT_TG_ARN>
terraform import module.alb.aws_lb_listener.https <DEV_HTTPS_LISTENER_ARN>
terraform import module.alb.aws_lb_listener.http_redirect <DEV_HTTP_LISTENER_ARN>
terraform import module.alb.aws_lb_listener_rule.api <DEV_API_RULE_ARN>   # priority-100 /api/*
# Client SPA rules (priorities 10–14 → client TG). Look up ARNs with
# `aws elbv2 describe-rules --listener-arn <HTTPS_LISTENER_ARN>`.
terraform import 'module.alb.aws_lb_listener_rule.extra["10"]' <RULE_ARN_PRIO_10>   # /
terraform import 'module.alb.aws_lb_listener_rule.extra["11"]' <RULE_ARN_PRIO_11>   # /assets/*
terraform import 'module.alb.aws_lb_listener_rule.extra["12"]' <RULE_ARN_PRIO_12>   # /oauth/*
terraform import 'module.alb.aws_lb_listener_rule.extra["13"]' <RULE_ARN_PRIO_13>   # /chats
terraform import 'module.alb.aws_lb_listener_rule.extra["14"]' <RULE_ARN_PRIO_14>   # /instance.json
```

Look up the ARNs with `aws elbv2 describe-load-balancers --names dev-e2eetext`, `describe-target-groups`, `describe-listeners`, and `describe-rules` (rule ARNs are full ARNs, usable directly as the import ID). After importing ALB/EC2/RDS, also import target-group attachments (AWS provider ≥ 6.34) with `terraform import module.alb.aws_lb_target_group_attachment.client '<tg-arn>,<instance-id>,8080'` and the same for `.server` with port `8081`. Everything else must be diff-free — the same acceptance bar as Phase 1. If plan shows anything else (drift since then), fix the `.tf`/`.tfvars` to match reality rather than accepting the diff; don't `apply` until it's clean.

Known, deliberate gaps for follow-up (each currently unmanaged by Terraform, so plan stays clean either way): the absent IAM instance profile (the "ECR pull via instance profile" setup this README documents — flip `create_ec2_iam_instance_profile = true` to migrate); and tags (defaults are `{}` to match the untagged live stack — set `tags`/`name_prefix` post-import to start tagging).

### `envs/prod` (Phase 2) — live status not yet confirmed

Unlike dev, whether a live prod EC2/ALB/VPC stack already exists has **not** been confirmed as part of this issue. This README's "Production on AWS" walkthrough only documents a procedure — every value in it (account ID, region, domain, VPC/subnet/security-group IDs, ACM cert ARN) is an obvious placeholder, not a recorded real identifier, so the repo alone doesn't prove a prod stack has actually been stood up.

Before running anything against `terraform/envs/prod`, confirm which situation applies:

```bash
aws ec2 describe-instances --filters "Name=tag:Name,Values=e2eetext-prod*"
aws elbv2 describe-load-balancers --names e2eetext-prod
```

- **If real resources come back:** follow the same import bootstrap as dev above, substituting prod's real identifiers.
- **If nothing comes back:** `terraform apply` can create the stack fresh — but get sign-off on instance size, domain, and ACM certificate choices first, since this would be standing up production infrastructure for real users.

### Phase 3 — RDS: not yet imported

`terraform/modules/rds` and its wiring in `terraform/envs/{dev,prod}` are written, `terraform fmt`/`validate` clean, but **not yet imported against either environment** — this phase was implemented with no AWS credentials available, so unlike dev's EC2/ALB/VPC stack in Phase 2, nothing about either environment's live RDS instance could actually be audited. Phase 2's dev audit did turn up circumstantial evidence a live dev RDS instance exists (the dev EC2 instance's hand-made `ec2-rds-1` security group, whose own description warns that detaching it can cut RDS connectivity), but that's not a substitute for reading the instance's real config.

Before running `terraform import` or `terraform apply` against either environment's `rds` module, audit the live instance (if one exists) and fill in `terraform.tfvars` accordingly:

```bash
aws rds describe-db-instances \
  --query 'DBInstances[].[DBInstanceIdentifier,EngineVersion,DBInstanceClass,AllocatedStorage,StorageType,StorageEncrypted,MasterUsername,DBName,MultiAZ,PubliclyAccessible,BackupRetentionPeriod,PreferredBackupWindow,PreferredMaintenanceWindow,DBSubnetGroup.DBSubnetGroupName,VpcSecurityGroups]'
aws rds describe-db-parameter-groups
aws rds describe-db-subnet-groups
```

- **If real resources come back:** fill in `terraform/envs/<env>/terraform.tfvars` with the audited values (see `terraform.tfvars.example` in each env for the full list and an `aws rds describe-db-instances` one-liner), pass the real master password out-of-band (`TF_VAR_rds_password`, never in `.tfvars`), confirm `terraform plan` is zero-diff, then import:

  ```bash
  cd terraform/envs/dev   # or prod
  terraform init

  terraform import 'module.rds.aws_db_subnet_group.this[0]' <SUBNET_GROUP_NAME>   # only if rds_create_subnet_group = true (default) — see below if the live instance uses the account's implicit "default" group
  terraform import module.rds.aws_security_group.this[0] <RDS_SECURITY_GROUP_ID>   # only if letting the module create/manage the SG
  terraform import 'module.rds.aws_db_parameter_group.this[0]' <PARAMETER_GROUP_NAME>  # only if rds_create_parameter_group = true
  terraform import module.rds.aws_db_instance.this <DB_INSTANCE_IDENTIFIER>
  ```

  If `aws rds describe-db-subnet-groups` shows the live instance in the account's implicit **default** DB subnet group (name literally `default` — every default VPC has one), set `rds_subnet_group_name = "default"` and `rds_create_subnet_group = false`, and skip the `aws_db_subnet_group` import above entirely — the AWS provider refuses to *create* a group named `default` (`"Default" is not allowed as "name"`), since it isn't a resource Terraform can own, only reference by name. This surfaced as a real `terraform plan (dev)` failure once live tfvars were configured against dev (PR #32).

  If `describe-db-instances` shows **`DBName` null** (instance created with no initial database), leave `rds_db_name` unset / `null`. Setting a name is ForceNew and will propose destroy/recreate of the live instance (observed on live-dev after import). Put the real PostgreSQL database the app uses in `rds_app_database_name` instead (live-dev: `dev_e2eetext`) — that attribute is not ForceNew.

  If the live instance's parameter group does not yet have `pg_cron` configured (i.e. #20 hasn't been done manually), applying after import will propose the `shared_preload_libraries=pg_cron` / `cron.database_name` changes — both are static parameters (`apply_method = "pending-reboot"`), so **the instance needs a reboot after `apply`** for them to take effect, and that reboot must happen before migration `000006` (which runs `CREATE EXTENSION pg_cron`) is ever applied against this instance — the same ordering warning #20's plan calls out.
- **If nothing comes back:** `terraform apply` can create the instance fresh — but get sign-off on instance class/storage/Multi-AZ choices first, same as any other production infrastructure decision, and doubly so for a database holding real user data.

### CI

`.github/workflows/terraform.yml` runs on PRs touching `terraform/`: `terraform fmt -check` always (whole tree), and `terraform init -backend=false` + `terraform validate` for each of `envs/shared`, `envs/dev`, `envs/prod` (matrixed — validate doesn't need variable values, so this needs no secrets). A `terraform plan` (never `apply`, given the blast radius) runs too, matrixed the same way, but only once repo secrets are configured and only against the state already bootstrapped above. The plan job deliberately uses its own `AWS_TERRAFORM_PLAN_ROLE_ARN` secret rather than reusing `build-images.yml`'s `AWS_ROLE_ARN` — that role is scoped to ECR push only (matching the manually-created policy exactly, for a zero-diff import) and does not have the IAM/ECR/S3 read access `terraform plan` needs. Provision a separate, read-only `terraform-plan` IAM role/OIDC trust and set it as `AWS_TERRAFORM_PLAN_ROLE_ARN` to enable this job. The `AWS_TERRAFORM_PLAN_ROLE_ARN`/`AWS_REGION` credentials step keeps `continue-on-error` so a misconfigured/under-scoped role doesn't abort the job on that one step alone, but the job as a whole is **not** allowed to report green when `terraform plan` doesn't actually run: a final "Fail if terraform plan did not run" step fails the job whenever `AWS_TERRAFORM_PLAN_ROLE_ARN`/`AWS_REGION`, or (for dev/prod) `DEV_TFVARS`/`PROD_TFVARS`, aren't configured — a skipped plan on a live-infra PR should block a reviewer's attention, not hide behind a green check (owner feedback on PR #32). Once tfvars *and* credentials are both present, `terraform init`/`terraform plan` themselves run **without** `continue-on-error` (PR #32 review) — a real failure there (bad tfvars, an under-scoped role, a genuine config error) fails the check the same way. The `plan` job also runs under the `terraform-plan` GitHub Environment — create it in repo Settings > Environments (ideally with required reviewers) so that, once real AWS credentials are wired up, a same-repo PR can't abuse edits to the workflow file itself to run arbitrary steps with those credentials ahead of review.

`DEV_TFVARS` / `PROD_TFVARS` are read with an explicit three-way expression (`matrix.environment == 'dev' && secrets.DEV_TFVARS || (matrix.environment == 'prod' && secrets.PROD_TFVARS || '')`), not the shorter `dev && X || Y` form — the shorter form treats an empty/unset `DEV_TFVARS` as falsy and falls through to `secrets.PROD_TFVARS`, which would plan **dev** state using **prod's** tfvars (including prod's RDS identity). This bug was introduced in Phase 2 and fixed there; PR #32's review caught the same bug reintroduced for the `dev`/`prod` RDS block and it's now fixed in both places the same way.

The live RDS master password is intentionally **not** part of `DEV_TFVARS`/`PROD_TFVARS` (each `rds_password` variable's own description already says never put it in a `.tfvars` file) — it's sourced from its own dedicated `DEV_RDS_PASSWORD` / `PROD_RDS_PASSWORD` secret instead, exported as `TF_VAR_rds_password` only for the `terraform plan` step. This keeps the full tfvars-blob secret shareable/rotatable without also carrying the database password.

CI plans run with `-lock=false`: the job never writes state, but Terraform's native S3 locking (`use_lockfile`) wants to `PutObject` a `.tflock` object even for a plan, which a read-only role must not be allowed to do (this surfaced as `Error acquiring the state lock` / `s3:PutObject AccessDenied` the first time the plan job ran with real credentials). Skipping the lock is safe precisely because CI never applies. The plan role is managed in `terraform/modules/terraform-plan-role` (wired from `envs/shared`); this is the policy document that module applies (state reads for every `envs/*` key; ECR/IAM reads for Phase 1 OIDC plus dig/prod EC2 instance profiles; EC2/ELBv2/ACM describes for Phase 2; RDS describes for Phase 3; SSM document/association reads for boot deploy):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "TerraformStateBucket",
      "Effect": "Allow",
      "Action": ["s3:ListBucket", "s3:GetBucketVersioning", "s3:GetBucketLocation"],
      "Resource": "arn:aws:s3:::e2eetext-terraform-state"
    },
    {
      "Sid": "TerraformStateReadOnly",
      "Effect": "Allow",
      "Action": ["s3:GetObject"],
      "Resource": "arn:aws:s3:::e2eetext-terraform-state/envs/*/terraform.tfstate"
    },
    {
      "Sid": "ECRRead",
      "Effect": "Allow",
      "Action": [
        "ecr:DescribeRepositories", "ecr:ListTagsForResource", "ecr:GetLifecyclePolicy",
        "ecr:GetRepositoryPolicy", "ecr:DescribeImageScanFindings"
      ],
      "Resource": [
        "arn:aws:ecr:*:<AWS_ACCOUNT_ID>:repository/e2eetext-server",
        "arn:aws:ecr:*:<AWS_ACCOUNT_ID>:repository/e2eetext-client"
      ]
    },
    {
      "Sid": "IAMReadManaged",
      "Effect": "Allow",
      "Action": [
        "iam:GetRole", "iam:GetRolePolicy", "iam:ListRolePolicies", "iam:ListAttachedRolePolicies",
        "iam:GetInstanceProfile", "iam:ListInstanceProfilesForRole",
        "iam:GetPolicy", "iam:GetPolicyVersion", "iam:ListPolicyVersions",
        "iam:GetOpenIDConnectProvider", "iam:ListOpenIDConnectProviders",
        "iam:ListRoleTags", "iam:ListPolicyTags", "iam:ListInstanceProfileTags"
      ],
      "Resource": [
        "arn:aws:iam::<AWS_ACCOUNT_ID>:role/github-actions-ecr-role",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:policy/github-actions-ecr-role",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:policy/github-actions-ecr-push",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:oidc-provider/token.actions.githubusercontent.com",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:role/terraform-plan-role",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:role/e2eetext-dev-ec2-role",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:role/e2eetext-prod-ec2-role",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:instance-profile/e2eetext-dev-ec2-profile",
        "arn:aws:iam::<AWS_ACCOUNT_ID>:instance-profile/e2eetext-prod-ec2-profile",
        "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
        "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
      ]
    },
    {
      "Sid": "IAMListAccount",
      "Effect": "Allow",
      "Action": ["iam:ListRoles", "iam:ListPolicies", "iam:ListInstanceProfiles"],
      "Resource": "*"
    },
    {
      "Sid": "NetworkAlbEc2AcmRead",
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*", "elasticloadbalancing:Describe*",
        "acm:DescribeCertificate", "acm:ListCertificates", "acm:ListTagsForCertificate"
      ],
      "Resource": "*"
    },
    {
      "Sid": "RdsRead",
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances", "rds:DescribeDBSubnetGroups",
        "rds:DescribeDBParameterGroups", "rds:DescribeDBParameters",
        "rds:ListTagsForResource"
      ],
      "Resource": "*"
    },
    {
      "Sid": "SsmBootDeployRead",
      "Effect": "Allow",
      "Action": [
        "ssm:DescribeDocument", "ssm:DescribeDocumentPermission",
        "ssm:ListTagsForResource", "ssm:GetDocument",
        "ssm:DescribeAssociation", "ssm:ListAssociations",
        "ssm:DescribeAssociationExecutions"
      ],
      "Resource": [
        "arn:aws:ssm:*:<AWS_ACCOUNT_ID>:document/e2eetext-boot-deploy",
        "arn:aws:ssm:*:<AWS_ACCOUNT_ID>:document/e2eetext-*-boot-deploy",
        "arn:aws:ssm:*:<AWS_ACCOUNT_ID>:association/*"
      ]
    }
  ]
}
```

## Containerized Claude ticket-processing runner

`maintenance/claude-runner/` packages the recurring "GitHub ticket processing" cycle described in `CLAUDE.md` (list changed issues/PRs, dispatch one agent per ticket, report) into a self-contained Docker image that runs unattended in a loop instead of needing someone to type the cycle prompt interactively.

```
maintenance/claude-runner/
├── Dockerfile          # node:22-bookworm-slim + go, docker CLI, aws-cli, git, curl, the claude-code npm package
├── entrypoint.sh        # clone/fetch dev, run `claude -p` on a timer, exec passthrough for ad hoc commands
├── docker-compose.yml   # standalone compose service (not the app's root docker-compose.yml)
└── .env.example          # runtime config template
```

Build and run:

```bash
cp maintenance/claude-runner/.env.example maintenance/claude-runner/.env
# edit maintenance/claude-runner/.env: GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN, GIT_AUTHOR_EMAIL, ...

docker compose -f maintenance/claude-runner/docker-compose.yml up -d --build
docker compose -f maintenance/claude-runner/docker-compose.yml logs -f
```

`docker run --rm <image> <cmd>` (or any command passed to the container) is exec'd directly instead of starting the loop — e.g. `docker run --rm claude-runner claude --version` — which is how the image itself is smoke-tested. With no command, the entrypoint clones/fetches the repo into a persistent `/workspace` volume and runs `claude -p "Run the GitHub issue/PR ticket-processing cycle described in CLAUDE.md."` every `CYCLE_INTERVAL_SECONDS` (default 300), logging and retrying on the next interval if a cycle fails rather than crash-looping the container.

The container runs with `docker.sock` mounted from the host (Docker-outside-of-Docker, not a nested `dockerd`) so `maintenance/ecr/build-and-push.sh` can build/push images using the host's Docker daemon and build cache — this grants the container host-root-equivalent access via that socket, which is acceptable only because this is the repo owner's own trusted automation on their own host.

### Permissions

Claude Code needs to run non-interactively (no "may I run this?" prompt blocking the loop), which this repo handles two ways:

- **`.claude/settings.json`** (committed, not a secret — it's tool-call patterns, not credentials) carries a curated `permissions.allow` allowlist for the `gh`/`git`/`go`/`npm`/`docker`/`aws`/`terraform`/`maintenance/*` commands the ticket-processing workflow actually uses, plus `worktree.bgIsolation: "worktree"` (per-agent isolated git worktrees) and `attribution.commit: ""` (no commit-message attribution trailer). `.claude/settings.local.json` stays gitignored for personal/machine-specific overrides — it can and does accumulate literal secrets (e.g. exact `curl -H "Authorization: Bearer <token>" ...` invocations get allow-listed verbatim), which is exactly why only the generalized, secret-free patterns from it were promoted into the committed file, not its content wholesale.
- `entrypoint.sh` runs `claude -p` with `--permission-mode dontAsk` **by default**, which relies on that allowlist rather than disabling the permission system. `CLAUDE_PERMISSION_MODE=bypassPermissions` (`--dangerously-skip-permissions`) is available as an explicit, owner-chosen opt-in escape hatch via the `.env` file if the allowlist ever proves too narrow for some new ticket shape — it is never the container's default.

`envs/dev`'s network/ec2/alb variables still default to the audited live values, and it looks up the ALB security group / ACM cert from the live ALB by name — but as of Phase 3, its `rds_*` variables (`rds_identifier`, `rds_engine_version`, `rds_instance_class`, ...) are required with no default, the same as `envs/prod`'s `ami_id`/`instance_type`/`alb_security_group_id`/`acm_certificate_arn` (plus prod's own `rds_*` set) — deliberately not defaulted, since neither environment's live RDS status was confirmed while writing Phase 3, and a wrong guess for an import target risks a forced replace on live resources. Both environments now need their tfvars supplied for `terraform plan` to run at all: the plan job reads them from the optional `DEV_TFVARS` / `PROD_TFVARS` secrets (full `.tfvars`-file contents, written to a gitignored `ci.auto.tfvars` before planning); until the relevant secret is populated — which for dev's `rds_*` values requires first auditing the live dev RDS instance, and for prod requires first confirming prod's real, live resource identifiers, per the sections above — that environment's `terraform plan` step is skipped, the same way the unconfigured-`AWS_TERRAFORM_PLAN_ROLE_ARN` case is — and, per the CI section above, the job now fails rather than reporting green in that case.
