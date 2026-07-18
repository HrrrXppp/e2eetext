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

export function formatDisappearCountdown(createdAt: string, disappearAfterMinutes: number, now = Date.now()): string {
  const msLeft = messageExpiresAt(createdAt, disappearAfterMinutes).getTime() - now;
  if (msLeft <= 0) {
    return "expired";
  }

  const minutesLeft = Math.ceil(msLeft / 60_000);
  if (minutesLeft < 60) {
    return `disappears in ${minutesLeft}m`;
  }

  const hoursLeft = Math.ceil(minutesLeft / 60);
  if (hoursLeft < 48) {
    return `disappears in ${hoursLeft}h`;
  }

  const daysLeft = Math.ceil(hoursLeft / 24);
  return `disappears in ${daysLeft}d`;
}
