import { expect, test, type Page } from "@playwright/test";

// The client shows a one-time "back up your private key" popup the first
// time a fresh identity keypair is generated (see IdentityBackupPrompt.tsx).
// Its backdrop is a full-viewport, click-intercepting overlay, so any test
// that continues interacting with the page after a fresh sign-in must
// dismiss it first if it appears. Not every sign-in path necessarily
// triggers fresh identity generation, so this is a short, best-effort wait
// rather than a hard assertion that it always shows up.
async function dismissIdentityBackupPromptIfPresent(page: Page): Promise<void> {
  const skipButton = page.getByRole("button", { name: "Skip for now" });
  try {
    await skipButton.waitFor({ state: "visible", timeout: 3_000 });
  } catch {
    return;
  }
  await skipButton.click();
  await expect(skipButton).toBeHidden();
}

// Signs in through the real OAuth authorization-code flow against the
// mock-Apple provider (see e2e/global-setup.ts and mockoidc/main.go's
// /apple mode): response_mode=form_post, private_key_jwt client
// authentication, and an ID token that never carries a name claim.
async function signInWithAppleE2E(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Sign in" }).click();
  await page.getByRole("button", { name: "Sign in with AppleE2E", exact: true }).click();
  await page.waitForURL("**/chats");
  await dismissIdentityBackupPromptIfPresent(page);
}

test.describe("Apple-like sign-in (response_mode=form_post + private_key_jwt + one-time name)", () => {
  test("populates the display name once from the one-time user payload, and a second sign-in still works without it", async ({
    browser,
  }) => {
    // A single browser context, matching the mock's own session cookie
    // (set by mockoidc's /apple/authorize on the browser making the
    // request) so the SECOND sign-in below is recognized as the same mock
    // subject — exactly like a real person signing back into a real IdP
    // without clearing cookies.
    const context = await browser.newContext();

    try {
      const page = await context.newPage();
      await page.goto("/");

      // First sign-in: mockoidc's /apple/authorize sends its one-time
      // "user" form field (name + email) only now — the real Apple
      // behavior this mode emulates — and the minted ID token itself never
      // carries a name claim at all, so this is the only source of the
      // display name.
      await signInWithAppleE2E(page);

      const nameButton = page.getByTitle("Edit display name");
      await expect(nameButton).toBeVisible();
      await expect(nameButton).not.toHaveText("Add name");
      const firstSignInName = (await nameButton.textContent())?.trim();
      expect(firstSignInName).toBeTruthy();

      // Sign out (clears only the app's own local session — the mock
      // identity provider's session cookie on 127.0.0.1:9998 is untouched,
      // same as real life).
      await page.getByRole("button", { name: "Sign out" }).click();
      await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

      // Second sign-in, same browser context/mock session: mockoidc
      // recognizes the same subject and does NOT resend the one-time user
      // payload this time. The previously-saved name must still be there,
      // and sign-in must still succeed (not silently log the user back out
      // for lack of a name in the ID token or the one-time payload).
      await signInWithAppleE2E(page);

      await expect(page.getByTitle("Edit display name")).toHaveText(firstSignInName ?? "");
    } finally {
      await context.close();
    }
  });
});
