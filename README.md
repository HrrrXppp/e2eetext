# Messenger

Monorepo messenger (pre-MVP) with a Go HTTP API and a React frontend. **Message encryption is not implemented yet** — payloads are stored in PostgreSQL as plaintext until the crypto layer lands. Authentication uses OIDC providers stored in PostgreSQL (Google is seeded by default).

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

Keep `client/package.json` `version` in sync with `VERSION` for npm metadata.

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
| `GET` | `/api/v1/chat?user_id=` | List chats for a user (scoped `user_id`; response `id` is scoped) |
| `POST` | `/api/v1/chat` | Create chat (`name`, `users_uids` with scoped user IDs) |
| `GET` | `/api/v1/message?chat_id=` | List messages in a chat (scoped `chat_id`; response `chatId` is scoped) |
| `POST` | `/api/v1/message` | Create message (`chat_id`, `user_id`, `data`; scoped IDs) |
| `PATCH` | `/api/v1/message/{nodeId}/{localId}` | Mark message read (`{"unread": false}`) |
| `GET` | `/api/v1/ws` | WebSocket for realtime events (`chat.added`, `chat.unread`) |

Scoped resource IDs use the form `{nodeId}/{localId}` (e.g. `99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111`).

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
