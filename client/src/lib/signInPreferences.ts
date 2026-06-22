const SKIP_PROFILE_KEY = "messenger_skip_profile_on_create";

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
