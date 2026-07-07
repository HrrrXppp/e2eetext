import { afterEach, describe, expect, it, vi } from "vitest";
import {
  canSearchUsers,
  createUser,
  ensureUserRegistered,
  fetchUsers,
  searchUsers,
  updateUserName,
  userDisplayName,
  userIdFallbackLabel,
  userLabel,
} from "@/lib/users";

const sampleUser = {
  id: "11111111-1111-1111-1111-111111111111",
  oidcProviderId: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  subject: "google-subject-1",
  name: "Test User",
  createdAt: "2026-06-10T12:00:00.000Z",
  updatedAt: "2026-06-10T12:00:00.000Z",
};

const claims = {
  subject: "google-subject-1",
  name: "Test User",
  provider: "google",
};

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("userDisplayName", () => {
  it("returns the stored name when present", () => {
    expect(userDisplayName("Test User")).toBe("Test User");
  });

  it("returns Add name when name is empty", () => {
    expect(userDisplayName(undefined)).toBe("Add name");
    expect(userDisplayName("")).toBe("Add name");
    expect(userDisplayName("   ")).toBe("Add name");
  });
});

describe("userIdFallbackLabel", () => {
  it("formats scoped ids as nodePrefix...localSuffix", () => {
    expect(userIdFallbackLabel("99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111")).toBe(
      "99999999...111111111111",
    );
  });

  it("returns shortened segments when the id is not scoped", () => {
    expect(userIdFallbackLabel("11111111-1111-1111-1111-111111111111")).toBe(
      "11111111...111111111111",
    );
  });
});

describe("userLabel", () => {
  it("prefers the user name when present", () => {
    expect(userLabel("Alice", "node/local")).toBe("Alice");
  });

  it("falls back to the scoped user id", () => {
    expect(
      userLabel(undefined, "99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111"),
    ).toBe("99999999...111111111111");
  });
});

describe("canSearchUsers", () => {
  it("requires at least 3 characters", () => {
    expect(canSearchUsers("ab")).toBe(false);
    expect(canSearchUsers("ali")).toBe(true);
    expect(canSearchUsers("Ев")).toBe(false);
    expect(canSearchUsers("Евг")).toBe(true);
  });
});

describe("searchUsers", () => {
  it("searches users with a single query parameter", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [sampleUser],
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(searchUsers("Test User")).resolves.toEqual([sampleUser]);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/search?q=Test+User", {
      headers: { Authorization: "Bearer token" },
    });
  });
});

describe("fetchUsers", () => {
  it("requests users with required filters", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [sampleUser],
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      fetchUsers({
        subject: "google-subject-1",
        oidcProviderId: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
      }),
    ).resolves.toEqual([sampleUser]);

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/user?subject=google-subject-1&oidc_provider_id=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
      { headers: { Authorization: "Bearer token" } },
    );
  });
});

describe("updateUserName", () => {
  it("updates the user display name", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const updatedUser = { ...sampleUser, name: "New Name" };
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => updatedUser,
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(updateUserName(sampleUser.id, "New Name")).resolves.toEqual(updatedUser);

    expect(fetchMock).toHaveBeenCalledWith(`/api/v1/user/${sampleUser.id}`, {
      method: "PATCH",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name: "New Name" }),
    });
  });
});

describe("createUser", () => {
  it("sends the public key in the request body", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleUser,
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(createUser({ kemPublicKey: "test-public-key" })).resolves.toEqual(sampleUser);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/user", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ kem_public_key: "test-public-key" }),
    });
  });

  it("sends skip_profile in the request body when requested", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleUser,
    });
    vi.stubGlobal("fetch", fetchMock);

    await createUser({ skipProfile: true, kemPublicKey: "test-public-key" });

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/user", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ kem_public_key: "test-public-key", skip_profile: true }),
    });
  });
});

describe("ensureUserRegistered", () => {
  it("returns existing user without creating a new one", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [sampleUser],
      });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      ensureUserRegistered(claims, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
    ).resolves.toEqual(sampleUser);

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("creates user without profile fields when skipProfile is enabled", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ ...sampleUser, name: undefined }),
      });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      ensureUserRegistered(claims, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", {
        skipProfile: true,
      }),
    ).resolves.toEqual({ ...sampleUser, name: undefined });

    const [url, init] = fetchMock.mock.calls[1];
    expect(url).toBe("/api/v1/user");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body as string);
    expect(body.skip_profile).toBe(true);
    expect(typeof body.kem_public_key).toBe("string");
    expect(body.kem_public_key.length).toBeGreaterThan(0);
  });

  it("creates user when not found", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => sampleUser,
      });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      ensureUserRegistered(claims, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
    ).resolves.toEqual(sampleUser);

    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("loads existing user when create returns conflict", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [],
      })
      .mockResolvedValueOnce({
        status: 409,
        ok: false,
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [sampleUser],
      });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      ensureUserRegistered(claims, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
    ).resolves.toEqual(sampleUser);

    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("deduplicates concurrent ensureUserRegistered calls", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [sampleUser],
    });
    vi.stubGlobal("fetch", fetchMock);

    const providerId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    const [first, second] = await Promise.all([
      ensureUserRegistered(claims, providerId),
      ensureUserRegistered(claims, providerId),
    ]);

    expect(first).toEqual(sampleUser);
    expect(second).toEqual(sampleUser);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
