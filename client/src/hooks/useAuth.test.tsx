import { type ReactNode } from "react";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthProvider, useAuth } from "@/hooks/useAuth";
import { makeToken } from "@/test/makeToken";

const fetchAuthProviders = vi.fn();
const ensureUserRegistered = vi.fn();

vi.mock("@/lib/oidcProviders", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/oidcProviders")>();
  return {
    ...actual,
    fetchAuthProviders: (...args: unknown[]) => fetchAuthProviders(...args),
  };
});

vi.mock("@/lib/users", () => ({
  ensureUserRegistered: (...args: unknown[]) => ensureUserRegistered(...args),
}));

vi.mock("@/lib/e2ee/session", () => ({
  ensureIdentityKeys: vi.fn().mockResolvedValue({ publicKey: "pk", secretKey: "sk" }),
}));

vi.mock("@/lib/e2ee/storage", () => ({
  clearE2EEState: vi.fn(),
}));

function wrapper({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

describe("useAuth", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
    fetchAuthProviders.mockReset();
    ensureUserRegistered.mockReset();
  });

  it("loads signed-in user from stored token", async () => {
    const idToken = makeToken({
      sub: "google-subject-1",
      iss: "https://accounts.google.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    localStorage.setItem("messenger_id_token", idToken);
    localStorage.setItem("messenger_auth_provider", "google");

    fetchAuthProviders.mockResolvedValue([
      {
        id: "provider-1",
        name: "Google",
        link: "https://accounts.google.com",
        slug: "google",
      },
    ]);
    ensureUserRegistered.mockResolvedValue({
      id: "user-1",
      subject: "google-subject-1",
      name: "Alice",
      oidcProviderId: "provider-1",
      createdAt: "2026-06-11T12:00:00.000Z",
      updatedAt: "2026-06-11T12:00:00.000Z",
    });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    expect(result.current.user).toEqual({
      id: "user-1",
      subject: "google-subject-1",
      name: "Alice",
      provider: "google",
      oidcProviderId: "provider-1",
    });
  });

  it("resolves the provider from the stored auth provider slug, not a guess from the issuer", async () => {
    // Two non-Google providers share the same "not google.com" issuer
    // shape the client used to fall back on for slug-guessing (see
    // idToken.ts); only the explicitly-stored provider slug (set at
    // sign-in time from the server's callback redirect) can tell them
    // apart.
    const idToken = makeToken({
      sub: "apple-subject-1",
      iss: "http://127.0.0.1:9998/apple",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    localStorage.setItem("messenger_id_token", idToken);
    localStorage.setItem("messenger_auth_provider", "apple");

    fetchAuthProviders.mockResolvedValue([
      { id: "oidc-provider-id", name: "OIDC", link: "http://127.0.0.1:9998", slug: "oidc" },
      { id: "apple-provider-id", name: "Apple", link: "http://127.0.0.1:9998/apple", slug: "apple" },
    ]);
    ensureUserRegistered.mockResolvedValue({
      id: "user-1",
      subject: "apple-subject-1",
      name: "Ada Appleseed",
      oidcProviderId: "apple-provider-id",
      createdAt: "2026-06-11T12:00:00.000Z",
      updatedAt: "2026-06-11T12:00:00.000Z",
    });

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(ensureUserRegistered).toHaveBeenCalledWith(
      expect.anything(),
      "apple-provider-id",
      expect.anything(),
    );
    expect(result.current.user?.oidcProviderId).toBe("apple-provider-id");
  });

  it("clears user when session is missing", async () => {
    fetchAuthProviders.mockResolvedValue([]);

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });
    expect(result.current.user).toBeNull();
  });

  it("throws outside provider", () => {
    expect(() => renderHook(() => useAuth())).toThrow(
      "useAuth must be used within AuthProvider",
    );
  });

  it("signs out and updates display name", async () => {
    fetchAuthProviders.mockResolvedValue([]);

    const { result } = renderHook(() => useAuth(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    result.current.setDisplayName("Bob");
    expect(result.current.user).toBeNull();

    result.current.signOut();
    expect(localStorage.getItem("messenger_id_token")).toBeNull();
  });
});
