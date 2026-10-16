import { beforeEach, describe, expect, it, vi } from "vitest";

const storage = vi.hoisted(() => ({
  cacheChatKey: vi.fn(),
  fetchChatKeyWraps: vi.fn(),
  fetchIdentityPublicKey: vi.fn(),
  getCachedChatKey: vi.fn(),
  loadStoredIdentity: vi.fn(),
  saveStoredIdentity: vi.fn(),
  uploadIdentityKey: vi.fn(),
}));
const crypto = vi.hoisted(() => ({ unwrapChatKey: vi.fn() }));

vi.mock("@/lib/e2ee/storage", () => storage);
vi.mock("@/lib/e2ee/crypto", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/e2ee/crypto")>()),
  unwrapChatKey: crypto.unwrapChatKey,
}));

import { ensureChatKey } from "@/lib/e2ee/session";

const wrap = (keyId: string) => ({ keyId, wrap: { keyId } });

describe("ensureChatKey", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    storage.getCachedChatKey.mockResolvedValue(null);
    storage.loadStoredIdentity.mockResolvedValue({ id: "identity" });
    storage.cacheChatKey.mockResolvedValue(undefined);
    crypto.unwrapChatKey.mockResolvedValue(new Uint8Array([1]));
  });

  it("fetches key wraps once for concurrent calls with a cold cache", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("k1")]);
    const results = await Promise.all(
      Array.from({ length: 10 }, () => ensureChatKey("u", "c", "k1")),
    );
    expect(storage.fetchChatKeyWraps).toHaveBeenCalledTimes(1);
    expect(results.every((r) => r.keyId === "k1")).toBe(true);
  });

  it("fetches once for distinct key ids in the same chat", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("k1"), wrap("k2")]);
    const [a, b] = await Promise.all([ensureChatKey("u", "c", "k1"), ensureChatKey("u", "c", "k2")]);
    expect(storage.fetchChatKeyWraps).toHaveBeenCalledTimes(1);
    expect([a.keyId, b.keyId]).toEqual(["k1", "k2"]);
  });

  it("retries after a failed fetch", async () => {
    storage.fetchChatKeyWraps.mockRejectedValueOnce(new Error("boom")).mockResolvedValue([wrap("k1")]);
    await expect(ensureChatKey("u", "c", "k1")).rejects.toThrow("boom");
    await expect(ensureChatKey("u", "c", "k1")).resolves.toMatchObject({ keyId: "k1" });
    expect(storage.fetchChatKeyWraps).toHaveBeenCalledTimes(2);
  });
});
