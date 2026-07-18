import { describe, expect, it } from "vitest";
import {
  DEFAULT_DISAPPEAR_AFTER_MINUTES,
  filterActiveMessages,
  formatDisappearAt,
  formatDisappearCountdown,
  isMessageExpired,
  messageExpiresAt,
} from "@/lib/disappear";

describe("disappear helpers", () => {
  it("defaults to 60 days in minutes", () => {
    expect(DEFAULT_DISAPPEAR_AFTER_MINUTES).toBe(86400);
  });

  it("computes expiry from createdAt + minutes", () => {
    const createdAt = "2026-01-01T00:00:00.000Z";
    const expires = messageExpiresAt(createdAt, 60);
    expect(expires.toISOString()).toBe("2026-01-01T01:00:00.000Z");
  });

  it("filters expired messages", () => {
    const now = Date.parse("2026-01-02T00:00:00.000Z");
    const messages = [
      { id: "a", createdAt: "2026-01-01T00:00:00.000Z" },
      { id: "b", createdAt: "2026-01-01T23:30:00.000Z" },
    ];
    expect(filterActiveMessages(messages, 60, now).map((m) => m.id)).toEqual(["b"]);
    expect(isMessageExpired(messages[0].createdAt, 60, now)).toBe(true);
  });

  it("formats an absolute disappear timestamp", () => {
    const createdAt = "2026-01-01T00:00:00.000Z";
    expect(formatDisappearAt(createdAt, 60)).toMatch(/2026/);
  });

  it("formats countdown", () => {
    const createdAt = "2026-01-01T00:00:00.000Z";
    const now = Date.parse("2026-01-01T00:30:00.000Z");
    expect(formatDisappearCountdown(createdAt, 60, now)).toBe("disappears in 30m");
  });
});
