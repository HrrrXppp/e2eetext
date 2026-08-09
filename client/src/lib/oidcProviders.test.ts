import { afterEach, describe, expect, it, vi } from "vitest";
import {
  fetchAuthProviders,
  findProviderBySlug,
  resetAuthProvidersCache,
} from "@/lib/oidcProviders";
import type { OIDCProvider } from "@/hooks/useAuth";

const sampleProviders: OIDCProvider[] = [
  {
    id: "11111111-1111-1111-1111-111111111111",
    name: "Google",
    link: "https://accounts.google.com",
    slug: "google",
  },
  {
    id: "22222222-2222-2222-2222-222222222222",
    name: "GitHub",
    link: "https://github.com",
    slug: "github",
  },
];

afterEach(() => {
  resetAuthProvidersCache();
  vi.unstubAllGlobals();
});

describe("fetchAuthProviders", () => {
  it("returns providers from the API", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => sampleProviders,
      }),
    );

    await expect(fetchAuthProviders()).resolves.toEqual(sampleProviders);
    expect(fetch).toHaveBeenCalledWith("/api/v1/auth/providers");
  });

  it("throws when the API request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
      }),
    );

    await expect(fetchAuthProviders()).rejects.toThrow("failed to load auth providers");
  });

  it("deduplicates concurrent requests", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleProviders,
    });
    vi.stubGlobal("fetch", fetchMock);

    const [first, second] = await Promise.all([
      fetchAuthProviders(),
      fetchAuthProviders(),
    ]);

    expect(first).toEqual(sampleProviders);
    expect(second).toEqual(sampleProviders);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("findProviderBySlug", () => {
  it("finds a provider by slug", () => {
    expect(findProviderBySlug(sampleProviders, "google")).toEqual(sampleProviders[0]);
  });

  it("returns undefined when the provider is missing", () => {
    expect(findProviderBySlug(sampleProviders, "missing")).toBeUndefined();
  });
});
