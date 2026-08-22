import { expect, type Page } from "@playwright/test";

// The client shows a one-time "back up your private key" popup the first
// time a fresh identity keypair is generated (see IdentityBackupPrompt.tsx).
// Its backdrop is a full-viewport, click-intercepting overlay, so any test
// that continues interacting with the page after a fresh sign-in must
// dismiss it first if it appears. Not every sign-in path necessarily
// triggers fresh identity generation, so this is a short, best-effort wait
// rather than a hard assertion that it always shows up.
export async function dismissIdentityBackupPromptIfPresent(page: Page): Promise<void> {
  const skipButton = page.getByRole("button", { name: "Skip for now" });
  try {
    await skipButton.waitFor({ state: "visible", timeout: 3_000 });
  } catch {
    return;
  }
  await skipButton.click();
  await expect(skipButton).toBeHidden();
}
