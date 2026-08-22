import { expect, test, type Page } from "@playwright/test";
import { dismissIdentityBackupPromptIfPresent } from "./identity-backup-prompt";

// Signs in through the real OAuth authorization-code flow against the
// mock-Google-shaped provider (see e2e/global-setup.ts's "GoogleE2E" row and
// mockoidc/main.go's default /authorize mode): a plain 302-redirect
// authorize step, a static client_secret, and an ID token that (unlike
// Apple's) always carries a `name` claim directly.
async function signInWithGoogleE2E(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.getByRole("button", { name: "Sign in with GoogleE2E", exact: true }).click();
  await page.waitForURL("**/chats");
  await dismissIdentityBackupPromptIfPresent(page);
}

test.describe("Google-like sign-in (GET redirect + static client secret + ID-token name claim)", () => {
  test("authorizes a person through the provider and registers them with the name from the ID token", async ({
    page,
  }) => {
    await page.goto("/");

    // First-time authorization: mockoidc's default /authorize mints a fresh
    // random test subject and its /token response's ID token carries
    // name="E2E Test User" directly (no one-time out-of-band payload the
    // way Apple's mode needs) — CreateFromToken (server/internal/service/
    // user.go) uses that claim as the new user's initial display name, so
    // it must show up immediately with no further action from the person.
    await signInWithGoogleE2E(page);

    const nameButton = page.getByTitle("Edit display name");
    await expect(nameButton).toBeVisible();
    await expect(nameButton).toHaveText("E2E Test User");
  });

  test("a second authorization mints a distinct person, since the mock issues a fresh subject each time", async ({
    page,
  }) => {
    await page.goto("/");
    await signInWithGoogleE2E(page);
    await expect(page.getByTitle("Edit display name")).toBeVisible();

    // Sign out and authorize again in the same browser: unlike /apple (which
    // tracks a session cookie so a second authorize reuses the same mock
    // subject), the default /authorize mode mints a brand-new random
    // "e2e-<hex>" subject on every hit with no memory of prior sign-ins —
    // exactly like a real Google account picker letting someone authorize a
    // second, different account. That must register as a genuinely separate
    // person, not silently reuse the first one.
    await page.getByRole("button", { name: "Sign out" }).click();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

    await signInWithGoogleE2E(page);

    // Still the same fixed name (mockoidc always claims "E2E Test User"),
    // but this is a fresh registration under a new subject — the app must
    // not be confused into treating the two sign-ins as the same account.
    // We can't assert on a distinct user ID from the UI alone, but the
    // display-name editor must still be freshly populated (not stuck on
    // stale state) and sign-in must succeed cleanly a second time.
    await expect(page.getByTitle("Edit display name")).toHaveText("E2E Test User");
  });
});
