import { execFile, spawn, type ChildProcess } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, "..", "..");
const SERVER_DIR = path.join(REPO_ROOT, "server");
const CLIENT_DIR = path.join(REPO_ROOT, "client");
const MOCKOIDC_DIR = path.join(REPO_ROOT, "mockoidc");

const MOCK_OIDC_URL = "http://127.0.0.1:9998";
const SERVER_URL = "http://127.0.0.1:8080";
const CLIENT_URL = "http://127.0.0.1:5173";
const OAUTH_CLIENT_ID = "e2e-test-client";

async function waitForHttp(url: string, timeoutMs: number, label: string): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    try {
      const res = await fetch(url);
      if (res.status < 500) {
        return;
      }
    } catch {
      // not up yet, keep polling
    }
    if (Date.now() > deadline) {
      throw new Error(`timed out waiting for ${label} at ${url}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
}

// Spawned detached (own process group) so killAll can signal the whole
// group: `go run` execs a compiled binary as a child of itself, and
// signaling only the `go run` PID does not reliably reach that grandchild.
function spawnLogged(
  label: string,
  command: string,
  args: string[],
  options: { cwd: string; env?: NodeJS.ProcessEnv },
): ChildProcess {
  const child = spawn(command, args, {
    cwd: options.cwd,
    env: options.env ?? process.env,
    stdio: ["ignore", "pipe", "pipe"],
    detached: true,
  });
  child.stdout?.on("data", (chunk: Buffer) => process.stdout.write(`[${label}] ${chunk}`));
  child.stderr?.on("data", (chunk: Buffer) => process.stderr.write(`[${label}] ${chunk}`));
  return child;
}

async function killAll(children: ChildProcess[]): Promise<void> {
  for (const child of children) {
    if (child.pid && child.exitCode === null && child.signalCode === null) {
      try {
        process.kill(-child.pid, "SIGTERM");
      } catch {
        // process group already gone
      }
    }
  }
  await new Promise((resolve) => setTimeout(resolve, 1000));
  for (const child of children) {
    if (child.pid && child.exitCode === null && child.signalCode === null) {
      try {
        process.kill(-child.pid, "SIGKILL");
      } catch {
        // process group already gone
      }
    }
  }
}

// Seeds an "OIDC" provider pointing at our mockoidc server, so the client's
// sign-in dialog offers "Sign in with OIDC" alongside the production
// "Sign in with Google" entry seeded by migration 000001. The name must
// slugify (lowercase) to exactly "oidc": the client infers a signed-in
// token's provider slug client-side from the ID token's issuer — "google"
// for a google.com issuer, "oidc" as the generic fallback for anything else
// (see src/lib/idToken.ts) — and useAuth() logs the user right back out if
// that inferred slug doesn't match a known provider's slug.
async function seedMockProvider(databaseURL: string): Promise<void> {
  await execFileAsync("psql", [
    databaseURL,
    "-v",
    "ON_ERROR_STOP=1",
    "-c",
    `INSERT INTO oidc_providers (name, link) SELECT 'OIDC', '${MOCK_OIDC_URL}' WHERE NOT EXISTS (SELECT 1 FROM oidc_providers WHERE name = 'OIDC');`,
  ]);
}

// Boots the whole stack (mock identity provider, real Go server, real Vite
// client) against a real Postgres, so the Playwright suite drives an actual
// browser through the same OAuth + E2EE code paths a production user hits —
// no mocked fetches, no bypassed crypto. Requires E2E_DATABASE_URL to point
// at a reachable, disposable Postgres (see client/e2e/README.md).
export default async function globalSetup(): Promise<() => Promise<void>> {
  const databaseURL = process.env.E2E_DATABASE_URL;
  if (!databaseURL) {
    throw new Error(
      "E2E_DATABASE_URL is required to run the Playwright E2E suite.\n" +
        "Start a disposable Postgres and point the suite at it, e.g.:\n" +
        "  docker run -d --name e2eetext-e2e-db -e POSTGRES_USER=messenger -e POSTGRES_PASSWORD=messenger \\\n" +
        "    -e POSTGRES_DB=messenger -p 55433:5432 postgres:16-alpine\n" +
        '  E2E_DATABASE_URL="postgres://messenger:messenger@127.0.0.1:55433/messenger?sslmode=disable" npm run test:e2e',
    );
  }

  const tmpDir = mkdtempSync(path.join(tmpdir(), "e2eetext-e2e-"));
  const configPath = path.join(tmpDir, "config.json");
  writeFileSync(
    configPath,
    JSON.stringify({
      oauth_credentials: {
        // Key must match the seeded provider's slug ("oidc" — see
        // seedMockProvider below), not the mockoidc binary's own name.
        oidc: { client_id: OAUTH_CLIENT_ID, client_secret: "unused-in-tests" },
      },
    }),
  );

  const children: ChildProcess[] = [];

  const mockOIDC = spawnLogged(
    "mockoidc",
    "go",
    ["run", ".", "--addr", ":9998", "--base-url", MOCK_OIDC_URL],
    { cwd: MOCKOIDC_DIR },
  );
  children.push(mockOIDC);
  await waitForHttp(`${MOCK_OIDC_URL}/.well-known/openid-configuration`, 60_000, "mockoidc");

  const goServer = spawnLogged("server", "go", ["run", "./cmd/messenger"], {
    cwd: SERVER_DIR,
    env: {
      ...process.env,
      DATABASE_URL: databaseURL,
      CONFIG_PATH: configPath,
      SERVER_ADDR: ":8080",
    },
  });
  children.push(goServer);
  await waitForHttp(`${SERVER_URL}/health`, 60_000, "go server");

  await seedMockProvider(databaseURL);

  const vite = spawnLogged("vite", "npm", ["run", "dev", "--", "--port", "5173", "--strictPort"], {
    cwd: CLIENT_DIR,
  });
  children.push(vite);
  await waitForHttp(`${CLIENT_URL}/`, 60_000, "vite dev server");

  return async () => {
    await killAll(children);
  };
}
