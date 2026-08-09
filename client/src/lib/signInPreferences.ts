const SKIP_PROFILE_KEY = "messenger_skip_profile_on_create";
const PENDING_SIGN_IN_NAME_KEY = "messenger_pending_sign_in_name";

export function setSkipProfileOnCreate(skip: boolean): void {
  if (skip) {
    localStorage.setItem(SKIP_PROFILE_KEY, "1");
  } else {
    localStorage.removeItem(SKIP_PROFILE_KEY);
  }
}

export function consumeSkipProfileOnCreate(): boolean {
  const skip = localStorage.getItem(SKIP_PROFILE_KEY) === "1";
  localStorage.removeItem(SKIP_PROFILE_KEY);
  return skip;
}

// Some providers (Apple) hand us the user's display name only once,
// out-of-band from the ID token, at OAuth callback time (see
// AuthCallback.tsx). It must survive the full-page navigation to /chats, so
// it's stashed here and consumed exactly once by the next ensureUserRegistered
// call, the same way skip-profile is.
export function setPendingSignInName(name?: string): void {
  const trimmed = name?.trim();
  if (trimmed) {
    localStorage.setItem(PENDING_SIGN_IN_NAME_KEY, trimmed);
  } else {
    localStorage.removeItem(PENDING_SIGN_IN_NAME_KEY);
  }
}

export function consumePendingSignInName(): string | undefined {
  const name = localStorage.getItem(PENDING_SIGN_IN_NAME_KEY) ?? undefined;
  localStorage.removeItem(PENDING_SIGN_IN_NAME_KEY);
  return name;
}
