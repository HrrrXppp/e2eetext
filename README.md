# Messenger

Monorepo messenger (pre-MVP) with a Go HTTP API and a React frontend. **New chats and messages use end-to-end encryption (E2EE v1)** — the server stores wrapped chat keys and opaque ciphertext only. Authentication uses OIDC providers stored in PostgreSQL (Google is seeded by default).

## Stack

| Layer    | Technology                          |
|----------|-------------------------------------|
| Backend  | Go 1.25, stdlib `net/http`, pgx     |
| Frontend | React 19, TypeScript, Vite 8        |
| Database | PostgreSQL 16                       |
| Auth     | OIDC (Google), stateless ID tokens  |

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
│   └── ec2/               # single-EC2 production compose
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
    }
  }
}
```

Get credentials from [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials. Create an **OAuth 2.0 Client ID** (Web application).

**Authorized redirect URIs** — register every origin where the app runs (the server builds the callback URL from the request host):

- Local dev: `http://localhost:5173/api/v1/auth/google/callback`
- Production: `https://app.example.com/api/v1/auth/google/callback`

OAuth callbacks are derived from `X-Forwarded-Host` and `X-Forwarded-Proto` on each request (the Vite dev proxy sets these in local dev).

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
| `GET` | `/api/v1/auth/{provider}/login` | Start OIDC login (e.g. `google`) |
| `GET` | `/api/v1/auth/{provider}/callback` | OAuth callback; redirects to client with tokens |
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
       → Redirect to /oauth/callback#id_token=…
       → Client stores ID token in localStorage
       → Client GET /api/v1/user?subject=…&oidc_provider_id=…; POST /api/v1/user if missing (identity from token)
       → Subsequent API calls send Authorization: Bearer <id_token>
```

The server verifies the ID token against the provider issuer on each protected request and reads user claims (`sub`, `email`, `name`) from the token — no server-side sessions.

## Database

Schema is defined in a single migration:

```
server/internal/database/migrations/000001_init.up.sql
```

Tables:

- **oidc_providers** — `id`, `name`, `link`, `picture` (BYTEA)
- **users** — linked to an OIDC provider via `oidc_provider_id` + `subject`

Google is inserted as the default provider on first migration.

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
- **Phase 3 (planned):** RDS, same per-environment split.

```
terraform/
├── modules/
│   ├── ecr/            # the e2eetext-server / e2eetext-client repos
│   ├── github-oidc/    # OIDC provider + IAM role/policy for build-images.yml
│   ├── network/         # data-source lookup of the VPC/subnets to deploy into
│   ├── alb/              # ALB + target groups + listener rules (matches create-alb.example.sh)
│   └── ec2/              # instance + security group + IAM instance profile
└── envs/
    ├── shared/          # root config instantiating ecr + github-oidc (Phase 1)
    ├── dev/              # root config instantiating network + alb + ec2 for dev (Phase 2)
    └── prod/             # same, for prod (Phase 2)
```

`terraform/modules/network` is deliberately data-source-only (it never creates/modifies/destroys a VPC or subnets) — it looks up the account's default VPC and subnets unless overridden, since nothing in this repo's scripts or docs ever creates a custom VPC. `terraform/modules/alb` also does not create the ALB's own security group; like `create-alb.example.sh`, it takes an existing security group ID as input. The `alb` and `ec2` modules default to the topology `create-alb.example.sh` describes, but expose overrides (target-group names, idle timeout, listener default action, rule priority/patterns, pre-existing instance SGs, optional IAM instance profile) because the live, hand-made dev stack diverges from the script on all of those — `terraform/envs/dev`'s defaults record the audited live values.

### One-time backend bootstrap

State is stored in S3 with native S3 state locking (Terraform >= 1.10, no DynamoDB table needed). The bucket referenced in `terraform/envs/shared/versions.tf` (`e2eetext-terraform-state`) does not exist yet — create it once, with versioning enabled, before the first `terraform init`:

```bash
aws s3api create-bucket --bucket e2eetext-terraform-state --region us-east-1
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
```

Replace `<AWS_ACCOUNT_ID>`, `<ROLE_NAME>`, and `<POLICY_NAME>` with the real values (`aws sts get-caller-identity`, `aws iam list-roles` / `list-policies` if the exact names aren't already known). After importing all six resources, `terraform plan` should show **no changes** — that's the acceptance bar, proving the config matches what's already live rather than describing a divergent target state. Only once that's confirmed is `terraform apply` safe to run.

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
terraform import module.alb.aws_lb_listener_rule.api <DEV_API_RULE_ARN>   # the priority-100 /api/* rule
```

Look up the ARNs with `aws elbv2 describe-load-balancers --names dev-e2eetext`, `describe-target-groups`, `describe-listeners`, and `describe-rules` (rule ARNs are full ARNs, usable directly as the import ID). After importing, `terraform plan` should show **no changes except** two `aws_lb_target_group_attachment` creates — the AWS provider (5.x) cannot import target-group attachments, and `RegisterTargets` on an already-registered target/port is an idempotent no-op, so those two "creates" are safe. Everything else must be diff-free — the same acceptance bar as Phase 1; this exact import sequence was verified clean against the live stack on 2026-07-26. If plan shows anything else (drift since then), fix the `.tf`/`.tfvars` to match reality rather than accepting the diff; don't `apply` until it's clean.

Known, deliberate gaps for follow-up (each currently unmanaged by Terraform, so plan stays clean either way): the hand-made client routing rules at priorities 10–14 (`/`, `/assets/*`, `/oauth/*`, `/chats`, `/instance.json`); the absent IAM instance profile (the "ECR pull via instance profile" setup this README documents — flip `create_ec2_iam_instance_profile = true` to migrate); and tags (defaults are `{}` to match the untagged live stack — set `tags`/`name_prefix` post-import to start tagging).

### `envs/prod` (Phase 2) — live status not yet confirmed

Unlike dev, whether a live prod EC2/ALB/VPC stack already exists has **not** been confirmed as part of this issue. This README's "Production on AWS" walkthrough only documents a procedure — every value in it (account ID, region, domain, VPC/subnet/security-group IDs, ACM cert ARN) is an obvious placeholder, not a recorded real identifier, so the repo alone doesn't prove a prod stack has actually been stood up.

Before running anything against `terraform/envs/prod`, confirm which situation applies:

```bash
aws ec2 describe-instances --filters "Name=tag:Name,Values=e2eetext-prod*"
aws elbv2 describe-load-balancers --names e2eetext-prod
```

- **If real resources come back:** follow the same import bootstrap as dev above, substituting prod's real identifiers.
- **If nothing comes back:** `terraform apply` can create the stack fresh — but get sign-off on instance size, domain, and ACM certificate choices first, since this would be standing up production infrastructure for real users.

### CI

`.github/workflows/terraform.yml` runs on PRs touching `terraform/`: `terraform fmt -check` always (whole tree), and `terraform init -backend=false` + `terraform validate` for each of `envs/shared`, `envs/dev`, `envs/prod` (matrixed — validate doesn't need variable values, so this needs no secrets). A `terraform plan` (never `apply`, given the blast radius) runs too, matrixed the same way, but only once repo secrets are configured and only against the state already bootstrapped above. The plan job deliberately uses its own `AWS_TERRAFORM_PLAN_ROLE_ARN` secret rather than reusing `build-images.yml`'s `AWS_ROLE_ARN` — that role is scoped to ECR push only (matching the manually-created policy exactly, for a zero-diff import) and does not have the IAM/ECR/S3 read access `terraform plan` needs. Provision a separate, read-only `terraform-plan` IAM role/OIDC trust and set it as `AWS_TERRAFORM_PLAN_ROLE_ARN` to enable this job; until then (or if it's under-scoped) the job either skips cleanly or, thanks to `continue-on-error` on its AWS steps, degrades to a warning instead of a hard failure. The `plan` job also runs under the `terraform-plan` GitHub Environment — create it in repo Settings > Environments (ideally with required reviewers) so that, once real AWS credentials are wired up, a same-repo PR can't abuse edits to the workflow file itself to run arbitrary steps with those credentials ahead of review.

`envs/dev` plans with no variables at all — its defaults are the audited live values, and it looks up the ALB security group / ACM cert from the live ALB by name (so its plan does need real AWS credentials, which the plan job's role provides). `envs/prod` still needs its required-with-no-default variables (`ami_id`, `instance_type`, `alb_security_group_id`, `acm_certificate_arn`) supplied for `terraform plan` to run — deliberately not defaulted, since prod's live status is unconfirmed and a wrong guess for an import target risks a forced replace on live resources. The plan job reads them from the optional `PROD_TFVARS` secret (full `.tfvars`-file contents, written to a gitignored `ci.auto.tfvars` before planning; `DEV_TFVARS` remains supported for overrides but is no longer required); until that's populated — which requires first confirming prod's real, live resource identifiers per the sections above — the prod plan step degrades to a skipped no-op the same way the unconfigured-`AWS_TERRAFORM_PLAN_ROLE_ARN` case does.
