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
const MOCK_OIDC_APPLE_URL = `${MOCK_OIDC_URL}/apple`;
const SERVER_URL = "http://127.0.0.1:8080";
const CLIENT_URL = "http://127.0.0.1:5173";
const OAUTH_CLIENT_ID = "e2e-test-client";
const OAUTH_GOOGLE_CLIENT_ID = "e2e-test-google-client";
const OAUTH_APPLE_CLIENT_ID = "e2e-test-apple-client";
const OAUTH_APPLE_KEY_ID = "e2e-apple-key";

type AppleTestClientKey = {
  private_key_pem: string;
  issuer: string;
  algorithm: string;
};

// Fetches the EC keypair mockoidc generated at startup for its /apple mode
// (private half only) so the server's config.json can use it to sign real
// private_key_jwt client-authentication assertions that the mock verifies —
// exercising the real mintPrivateKeyJWT/oauthConfig code path against the
// mock, not a hardcoded/checked-in key.
async function fetchAppleTestClientKey(): Promise<AppleTestClientKey> {
  const res = await fetch(`${MOCK_OIDC_APPLE_URL}/test-client-key`);
  if (!res.ok) {
    throw new Error(`failed to fetch apple test client key: ${res.status}`);
  }
  return (await res.json()) as AppleTestClientKey;
}

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

// Seeds an "OIDC" provider pointing at our mockoidc server's default mode,
// so the client's sign-in dialog offers "Sign in with OIDC" alongside the
// production "Sign in with Google" entry seeded by migration 000001; a
// "GoogleE2E" provider pointing at that same default mode under its own
// name/slug, so a dedicated spec can exercise a person actually authorizing
// through a Google-shaped ("Sign in with GoogleE2E") button end-to-end
// without depending on golden-path.spec.ts's use of the "OIDC" one (kept
// distinct so the two specs' provider state can't accidentally couple); and
// a separate "AppleE2E" provider pointing at mockoidc's Apple-like /apple
// mode (see mockoidc/main.go and migration 000007's own production "Apple"
// row, which points at the real appleid.apple.com and is left untouched —
// this is a second, E2E-only row so the mock is exercised without needing
// real Apple credentials). The provider name must slugify (lowercase) to
// exactly the slug the client requests: useAuth() resolves the signed-in
// provider from the explicit slug threaded through the OAuth callback
// redirect (see src/app/AuthCallback.tsx), not by guessing from the ID
// token issuer, so distinct provider names/slugs are all that's required
// here — no special-casing needed for a second non-Google-shaped provider.
async function seedMockProviders(databaseURL: string): Promise<void> {
  await execFileAsync("psql", [
    databaseURL,
    "-v",
    "ON_ERROR_STOP=1",
    "-c",
    `INSERT INTO oidc_providers (name, link)
       SELECT 'OIDC', '${MOCK_OIDC_URL}'
       WHERE NOT EXISTS (SELECT 1 FROM oidc_providers WHERE name = 'OIDC');
     INSERT INTO oidc_providers (name, link)
       SELECT 'GoogleE2E', '${MOCK_OIDC_URL}'
       WHERE NOT EXISTS (SELECT 1 FROM oidc_providers WHERE name = 'GoogleE2E');
     INSERT INTO oidc_providers (name, link, scopes, response_mode, client_secret_strategy)
       SELECT 'AppleE2E', '${MOCK_OIDC_APPLE_URL}', 'openid name', 'form_post', 'private_key_jwt'
       WHERE NOT EXISTS (SELECT 1 FROM oidc_providers WHERE name = 'AppleE2E');`,
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

  const children: ChildProcess[] = [];

  // mockoidc must be up first: the server's config needs the EC private key
  // it generates at startup for /apple's private_key_jwt verification.
  const mockOIDC = spawnLogged(
    "mockoidc",
    "go",
    ["run", ".", "--addr", ":9998", "--base-url", MOCK_OIDC_URL],
    { cwd: MOCKOIDC_DIR },
  );
  children.push(mockOIDC);
  await waitForHttp(`${MOCK_OIDC_URL}/.well-known/openid-configuration`, 60_000, "mockoidc");

  const appleKey = await fetchAppleTestClientKey();
  writeFileSync(
    configPath,
    JSON.stringify({
      oauth_credentials: {
        // Key must match the seeded providers' slugs (see
        // seedMockProviders below), not the mockoidc binary's own name.
        oidc: { client_id: OAUTH_CLIENT_ID, client_secret: "unused-in-tests" },
        googlee2e: { client_id: OAUTH_GOOGLE_CLIENT_ID, client_secret: "unused-in-tests" },
        applee2e: {
          client_id: OAUTH_APPLE_CLIENT_ID,
          private_key_jwt_issuer: appleKey.issuer,
          private_key_jwt_key_id: OAUTH_APPLE_KEY_ID,
          private_key_jwt_algorithm: appleKey.algorithm,
          private_key_jwt_private_key: appleKey.private_key_pem,
        },
      },
    }),
  );

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

  await seedMockProviders(databaseURL);

  // Serve a real production build (vite preview) instead of `vite dev`.
  // The dev server's HMR client can trigger unprompted full-page reloads
  // (e.g. on WebSocket reconnect), which aborts whatever fetch the app
  // happens to have in flight at that moment — this showed up in CI as
  // GET /api/v1/user getting net::ERR_ABORTED mid-sign-in, immediately
  // followed by an unexplained extra page load in the trace. `preview`
  // serves static built assets with no HMR machinery at all, and is more
  // representative of what actually ships anyway.
  await execFileAsync("npm", ["run", "build"], { cwd: CLIENT_DIR });

  // --host 127.0.0.1 pins vite to the exact address CLIENT_URL polls below —
  // without it, vite's default "localhost" bind and Node's "localhost"
  // resolution can land on different loopback interfaces (IPv4 vs IPv6)
  // depending on the OS/Node version, causing waitForHttp to time out even
  // though the server is actually up (seen on GitHub Actions' Node 24
  // runners; not reproduced with every local Node version).
  const vite = spawnLogged(
    "vite",
    "npm",
    ["run", "preview", "--", "--host", "127.0.0.1", "--port", "5173", "--strictPort"],
    { cwd: CLIENT_DIR },
  );
  children.push(vite);
  await waitForHttp(`${CLIENT_URL}/`, 60_000, "vite preview");

  return async () => {
    await killAll(children);
  };
}
