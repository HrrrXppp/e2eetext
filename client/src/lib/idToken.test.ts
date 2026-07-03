import { describe, expect, it } from "vitest";
import { getTokenExpiration, isTokenExpired, parseIdToken } from "@/lib/idToken";
import { makeToken } from "@/test/makeToken";

describe("parseIdToken", () => {
  it("extracts subject and google provider", () => {
    const token = makeToken({
      sub: "google-subject-1",
      name: "Test User",
      iss: "https://accounts.google.com",
    });

    expect(parseIdToken(token)).toEqual({
      subject: "google-subject-1",
      name: "Test User",
      provider: "google",
      expiresAt: undefined,
    });
  });

  it("returns null for invalid token", () => {
    expect(parseIdToken("not-a-jwt")).toBeNull();
  });

  it("decodes utf-8 names from jwt payload", () => {
    const token = makeToken({
      sub: "google-subject-1",
      name: "Евгений Хрунов",
      iss: "https://accounts.google.com",
    });

    expect(parseIdToken(token)?.name).toBe("Евгений Хрунов");
  });
});

describe("getTokenExpiration", () => {
  it("reads exp from jwt payload", () => {
    const token = makeToken({
      sub: "google-subject-1",
      exp: 1_700_000_000,
    });

    expect(getTokenExpiration(token)).toBe(1_700_000_000);
  });
});

describe("isTokenExpired", () => {
  it("returns false before expiration", () => {
    const future = Math.floor(Date.now() / 1000) + 3600;
    const token = makeToken({ sub: "google-subject-1", exp: future });

    expect(isTokenExpired(token)).toBe(false);
  });

  it("returns true within skew window", () => {
    const soon = Math.floor(Date.now() / 1000) + 120;
    const token = makeToken({ sub: "google-subject-1", exp: soon });

    expect(isTokenExpired(token, 300)).toBe(true);
  });
});
