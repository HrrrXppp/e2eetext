import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  // Generous: each test drives two independently signed-in real browser
  // profiles through real OAuth + real Postgres + real ML-KEM/AES crypto,
  // not mocked fetches — a handful of real network round trips per step
  // adds up fast, especially on a loaded machine.
  timeout: 90_000,
  expect: { timeout: 15_000 },
  fullyParallel: false,
  workers: 1,
  // Real dev-proxy + real Postgres + real browser network stack means an
  // occasional dropped connection under load is infra noise, not a product
  // bug — retry once before failing the run.
  retries: 1,
  reporter: [["list"], ["html", { open: "never", outputFolder: "playwright-report" }]],
  globalSetup: "./e2e/global-setup.ts",
  use: {
    baseURL: "http://127.0.0.1:5173",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
});
