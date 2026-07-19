export const DEFAULT_DISAPPEAR_AFTER_MINUTES = 60 * 24 * 60; // 60 days

/** Expiry instant = message createdAt plus the chat's TTL in minutes. */
export function messageExpiresAt(createdAt: string, disappearAfterMinutes: number): Date {
  return new Date(new Date(createdAt).getTime() + disappearAfterMinutes * 60_000);
}

export function isMessageExpired(createdAt: string, disappearAfterMinutes: number, now = Date.now()): boolean {
  return messageExpiresAt(createdAt, disappearAfterMinutes).getTime() <= now;
}

export function filterActiveMessages<T extends { createdAt: string }>(
  messages: T[],
  disappearAfterMinutes: number,
  now = Date.now(),
): T[] {
  return messages.filter((message) => !isMessageExpired(message.createdAt, disappearAfterMinutes, now));
}

export function formatDisappearAt(createdAt: string, disappearAfterMinutes: number): string {
  return messageExpiresAt(createdAt, disappearAfterMinutes).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}
