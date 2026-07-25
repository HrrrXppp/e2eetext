import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { expect, test, type Page } from "@playwright/test";

const execFileAsync = promisify(execFile);

// Queries the most recently created message's raw `data` column directly
// from Postgres — the same database global-setup.ts pointed the whole
// stack at (E2E_DATABASE_URL) — so we assert on what the server actually
// persisted, not just what the sender's own browser rendered back to it.
async function readLatestMessageDataFromDB(): Promise<string> {
  const databaseURL = process.env.E2E_DATABASE_URL;
  if (!databaseURL) {
    throw new Error("E2E_DATABASE_URL is required (set by global-setup.ts)");
  }
  const { stdout } = await execFileAsync("psql", [
    databaseURL,
    "-t",
    "-A",
    "-c",
    "SELECT data FROM messages ORDER BY created_at DESC LIMIT 1;",
  ]);
  return stdout.trim();
}

// Signs in through the real OAuth authorization-code flow against the mock
// identity provider (see e2e/global-setup.ts), then sets a deterministic
// display name so the other browser context can find this user by name in
// the "new chat" search — mirroring how a real user would look someone up.
async function signIn(page: Page, displayName: string): Promise<void> {
  // Opt-in only (DEBUG_E2E=1) — logging this on every run, including
  // passing ones, would flood CI output for no benefit.
  if (process.env.DEBUG_E2E) {
    page.on("console", (msg) => console.log(`[console:${msg.type()}]`, msg.text()));
    page.on("pageerror", (err) => console.log("[pageerror]", err.message, err.stack));
    page.on("requestfailed", (req) => console.log("[requestfailed]", req.url(), req.failure()?.errorText));
    page.on("response", (res) => {
      if (res.url().includes("/api/v1/")) console.log("[response]", res.status(), res.url());
    });
  }
  await page.goto("/");
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.getByRole("button", { name: "Sign in with OIDC" }).click();
  await page.waitForURL("**/chats");

  await page.getByTitle("Edit display name").click();
  await page.getByLabel("Display name").fill(displayName);
  await page.getByRole("button", { name: "Save name" }).click();
  await expect(page.getByTitle("Edit display name")).toHaveText(displayName);
}

test("two real browsers sign in, create a chat, and exchange an E2EE message", async ({ browser }) => {
  const runID = `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`;
  const aliceName = `Alice E2E ${runID}`;
  const bobName = `Bob E2E ${runID}`;
  const chatName = `E2E chat ${runID}`;
  const plaintext = `hello from alice, run ${runID}`;

  const aliceContext = await browser.newContext();
  const bobContext = await browser.newContext();

  try {
    const alicePage = await aliceContext.newPage();
    const bobPage = await bobContext.newPage();

    // Each context is a separate, independently authenticated real browser
    // profile — its own localStorage, its own IndexedDB-backed ML-KEM
    // identity keypair, exactly like two different people on two devices.
    await signIn(alicePage, aliceName);
    await signIn(bobPage, bobName);

    await alicePage.getByRole("button", { name: "New chat" }).click();
    await alicePage.getByLabel("Chat name").fill(chatName);
    await alicePage.getByRole("button", { name: "Search users" }).click();
    await alicePage.getByPlaceholder("Name or user ID").fill(bobName);
    await alicePage.getByRole("button", { name: "Search", exact: true }).click();
    await alicePage.getByRole("button", { name: "Add" }).click();
    await alicePage.getByRole("button", { name: "Create chat" }).click();

    const aliceChatOption = alicePage.getByRole("option", { name: new RegExp(chatName) });
    await expect(aliceChatOption).toBeVisible();

    // exact: true — getByLabel does substring matching by default, and
    // "Message" is also a substring of the messages panel's own
    // aria-label ("Messages in {chatName}"), so a non-exact match resolves
    // to two elements (strict-mode violation).
    await alicePage.getByLabel("Message", { exact: true }).fill(plaintext);
    await alicePage.getByRole("button", { name: "Send" }).click();

    await expect(
      alicePage.locator(".chats-page__message-text").filter({ hasText: plaintext }),
    ).toBeVisible();

    // Verify what's actually stored in Postgres, not just what the sender's
    // own browser renders back — the server must never see the plaintext.
    const storedData = await readLatestMessageDataFromDB();
    expect(storedData.length).toBeGreaterThan(0);
    expect(storedData).not.toContain(plaintext);
    expect(storedData.toLowerCase()).not.toContain(plaintext.toLowerCase());

    // The real assertion: Bob's browser independently unwraps the chat key
    // (via his own ML-KEM secret key) and decrypts the message client-side.
    // The server only ever relayed ciphertext — if this renders, real E2EE
    // round-tripped across two separate browser identities.
    const bobChatOption = bobPage.getByRole("option", { name: new RegExp(chatName) });
    await expect(bobChatOption).toBeVisible({ timeout: 15_000 });
    await bobChatOption.click();

    await expect(
      bobPage.locator(".chats-page__message-text").filter({ hasText: plaintext }),
    ).toBeVisible({ timeout: 15_000 });
  } finally {
    await aliceContext.close();
    await bobContext.close();
  }
});
