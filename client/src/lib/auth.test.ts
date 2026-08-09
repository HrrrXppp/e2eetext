import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  authHeaders,
  clearIdToken,
  getAuthProvider,
  getBearerToken,
  getRefreshToken,
  getWebSocketIdToken,
  hasUsableSession,
  refreshAuthTokensIfNeeded,
  storeAuthSession,
} from "@/lib/auth";
import { makeToken } from "@/test/makeToken";

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("storeAuthSession", () => {
  it("stores id, refresh tokens, and provider", () => {
    const idToken = makeToken({
      sub: "google-subject-1",
      iss: "https://accounts.google.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });

    storeAuthSession({
      idToken,
      refreshToken: "refresh-token",
      accessToken: "access-token",
    });

    expect(getRefreshToken()).toBe("refresh-token");
    expect(getAuthProvider()).toBe("google");
  });
});

describe("refreshAuthTokensIfNeeded", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-14T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("skips refresh when token is still valid", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) + 3600,
      }),
      refreshToken: "refresh-token",
    });

    await expect(refreshAuthTokensIfNeeded()).resolves.toBe(true);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("refreshes token when exp is within skew window", async () => {
    const exp = Math.floor(Date.now() / 1000) + 120;
    const newIdToken = makeToken({
      sub: "google-subject-1",
      iss: "https://accounts.google.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });

    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp,
      }),
      refreshToken: "refresh-token",
    });

    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        idToken: newIdToken,
        accessToken: "new-access-token",
        refreshToken: "new-refresh-token",
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(refreshAuthTokensIfNeeded()).resolves.toBe(true);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        provider: "google",
        refreshToken: "refresh-token",
      }),
    });
    expect(getRefreshToken()).toBe("new-refresh-token");
  });

  it("returns false when refresh token is missing and session expired", async () => {
    clearIdToken();
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) - 10,
      }),
    });

    await expect(refreshAuthTokensIfNeeded()).resolves.toBe(false);
  });

  it("keeps existing refresh token when login omits it", () => {
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) + 3600,
      }),
      refreshToken: "existing-refresh-token",
    });

    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) + 7200,
      }),
      accessToken: "new-access-token",
    });

    expect(getRefreshToken()).toBe("existing-refresh-token");
  });
});

describe("getBearerToken", () => {
  it("uses access token when id token is expired", () => {
    const exp = Math.floor(Date.now() / 1000) - 10;
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp,
      }),
      accessToken: "access-token",
    });

    expect(getBearerToken()).toBe("access-token");
  });
});

describe("getWebSocketIdToken", () => {
  it("returns a valid id token", () => {
    const idToken = makeToken({
      sub: "google-subject-1",
      iss: "https://accounts.google.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    storeAuthSession({ idToken });

    expect(getWebSocketIdToken()).toBe(idToken);
  });

  it("returns null when only an access token is available", () => {
    localStorage.setItem("messenger_access_token", "ya29.access-token");

    expect(getWebSocketIdToken()).toBeNull();
  });

  it("returns null when the id token is expired", () => {
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) - 10,
      }),
      accessToken: "access-token",
    });

    expect(getWebSocketIdToken()).toBeNull();
  });
});

describe("authHeaders", () => {
  it("returns authorization header when token exists", () => {
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) + 3600,
      }),
    });

    expect(authHeaders()).toEqual({ Authorization: expect.stringMatching(/^Bearer /) });
  });

  it("returns empty headers without session", () => {
    clearIdToken();
    expect(authHeaders()).toEqual({});
  });
});

describe("hasUsableSession", () => {
  it("returns true for valid token", () => {
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) + 3600,
      }),
    });

    expect(hasUsableSession()).toBe(true);
  });

  it("returns false when token is expired", () => {
    storeAuthSession({
      idToken: makeToken({
        sub: "google-subject-1",
        iss: "https://accounts.google.com",
        exp: Math.floor(Date.now() / 1000) - 10,
      }),
    });

    expect(hasUsableSession()).toBe(false);
  });
});
