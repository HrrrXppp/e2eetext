import { afterEach, describe, expect, it, vi } from "vitest";
import {
  fetchInstanceConfig,
  isDevelopmentInstance,
  resetInstanceConfigCache,
} from "@/lib/instance";

describe("isDevelopmentInstance", () => {
  it("returns true when instance config is development", () => {
    expect(isDevelopmentInstance({ environment: "development" })).toBe(true);
  });

  it("returns false for production config", () => {
    expect(isDevelopmentInstance({ environment: "production" })).toBe(false);
  });

  it("returns false when config is missing", () => {
    expect(isDevelopmentInstance(null)).toBe(false);
  });
});

describe("fetchInstanceConfig", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    resetInstanceConfigCache();
  });

  it("loads environment from instance.json", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ environment: "development" }),
      }),
    );

    await expect(fetchInstanceConfig()).resolves.toEqual({ environment: "development" });
  });

  it("returns null when instance.json is missing", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
      }),
    );

    await expect(fetchInstanceConfig()).resolves.toBeNull();
  });
});
