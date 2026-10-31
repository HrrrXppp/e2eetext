import { beforeEach, describe, expect, it, vi } from "vitest";

const storage = vi.hoisted(() => ({
  cacheChatKey: vi.fn(),
  fetchChatKeyWraps: vi.fn(),
  fetchIdentityPublicKey: vi.fn(),
  getCachedChatKey: vi.fn(),
  loadStoredIdentity: vi.fn(),
  saveStoredIdentity: vi.fn(),
  uploadChatKeyVersion: vi.fn(),
  uploadIdentityKey: vi.fn(),
}));
const crypto = vi.hoisted(() => ({
  unwrapChatKey: vi.fn(),
  wrapChatKey: vi.fn(),
  encryptMessage: vi.fn(),
}));
const chats = vi.hoisted(() => ({ getChatMembers: vi.fn() }));

vi.mock("@/lib/chats", () => chats);

vi.mock("@/lib/e2ee/storage", () => storage);
vi.mock("@/lib/e2ee/crypto", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/e2ee/crypto")>()),
  unwrapChatKey: crypto.unwrapChatKey,
  wrapChatKey: crypto.wrapChatKey,
  encryptMessage: crypto.encryptMessage,
}));

import { ChatKeyUnavailableError, encryptOutgoingMessage, ensureChatKey, prepareAddMembers } from "@/lib/e2ee/session";

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

describe("key versions", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    storage.getCachedChatKey.mockResolvedValue(null);
    storage.loadStoredIdentity.mockResolvedValue({ publicKey: "mine", secretKey: "secret" });
    storage.cacheChatKey.mockResolvedValue(undefined);
    storage.uploadChatKeyVersion.mockResolvedValue(undefined);
    crypto.unwrapChatKey.mockResolvedValue(new Uint8Array([1]));
    crypto.wrapChatKey.mockImplementation(async ({ keyId }: { keyId: string }) => ({ keyId, wrapped: true }));
    crypto.encryptMessage.mockImplementation(async ({ keyId }: { keyId: string }) => ({
      v: 1,
      alg: "aes256gcm-chat-key",
      keyId,
      nonce: "n",
      ciphertext: "c",
    }));
  });

  it("decrypts with exactly the version named by the message", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("old"), wrap("new")]);
    await expect(ensureChatKey("u", "c", "old")).resolves.toMatchObject({ keyId: "old" });
    expect(crypto.unwrapChatKey).toHaveBeenCalledWith(expect.objectContaining({ wrap: { keyId: "old" } }));
  });

  it("refetches wraps for an unknown key id and finds the new version", async () => {
    storage.fetchChatKeyWraps
      .mockResolvedValueOnce([wrap("old")])
      .mockResolvedValueOnce([wrap("old"), wrap("new")]);
    await expect(ensureChatKey("u", "c", "new")).resolves.toMatchObject({ keyId: "new" });
    expect(storage.fetchChatKeyWraps).toHaveBeenCalledTimes(2);
  });

  it("does not fall back to another version when the message version is missing", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("old")]);
    await expect(ensureChatKey("u", "c", "missing")).rejects.toBeInstanceOf(ChatKeyUnavailableError);
    expect(crypto.unwrapChatKey).not.toHaveBeenCalled();
  });

  it("reports a wrap that the identity cannot open as unavailable", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("old")]);
    crypto.unwrapChatKey.mockRejectedValue(new Error("bad tag"));
    await expect(ensureChatKey("u", "c", "old")).rejects.toBeInstanceOf(ChatKeyUnavailableError);
  });

  it("keeps non-key errors distinct", async () => {
    storage.fetchChatKeyWraps.mockRejectedValue(new Error("network"));
    const error = await ensureChatKey("u", "c").catch((e: unknown) => e);
    expect(error).not.toBeInstanceOf(ChatKeyUnavailableError);
  });

  it("adds a new key version for every member and sends with it when the newest wrap cannot be opened", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("old")]);
    crypto.unwrapChatKey.mockRejectedValue(new Error("bad tag"));
    chats.getChatMembers.mockResolvedValue([{ id: "u" }, { id: "bob" }]);
    storage.fetchIdentityPublicKey.mockResolvedValue({ publicKey: "bobs" });
    vi.spyOn(console, "error").mockImplementation(() => undefined);

    const serialized = await encryptOutgoingMessage({ plaintext: "hi", chatId: "c", senderUserId: "u" });

    expect(storage.fetchIdentityPublicKey).toHaveBeenCalledTimes(1);
    expect(storage.fetchIdentityPublicKey).toHaveBeenCalledWith("bob");
    expect(storage.uploadChatKeyVersion).toHaveBeenCalledTimes(1);
    const [chatId, keyId, wraps] = storage.uploadChatKeyVersion.mock.calls[0];
    expect(chatId).toBe("c");
    expect(wraps.map((item: { userId: string }) => item.userId)).toEqual(["u", "bob"]);
    expect(keyId).not.toBe("old");
    expect(storage.cacheChatKey).toHaveBeenCalledWith("u", "c", keyId, expect.any(Uint8Array));
    expect(JSON.parse(serialized).keyId).toBe(keyId);
  });

  it("does not add a key version when the newest wrap opens", async () => {
    storage.fetchChatKeyWraps.mockResolvedValue([wrap("old"), wrap("new")]);
    const serialized = await encryptOutgoingMessage({ plaintext: "hi", chatId: "c", senderUserId: "u" });
    expect(storage.uploadChatKeyVersion).not.toHaveBeenCalled();
    expect(JSON.parse(serialized).keyId).toBe("new");
  });

  it("does not add a key version after a network failure", async () => {
    storage.fetchChatKeyWraps.mockRejectedValue(new Error("network"));
    await expect(
      encryptOutgoingMessage({ plaintext: "hi", chatId: "c", senderUserId: "u" }),
    ).rejects.toThrow("network");
    expect(storage.uploadChatKeyVersion).not.toHaveBeenCalled();
  });
});

describe("prepareAddMembers", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    storage.loadStoredIdentity.mockResolvedValue({ publicKey: "mine", secretKey: "secret" });
    storage.cacheChatKey.mockResolvedValue(undefined);
    chats.getChatMembers.mockResolvedValue([{ id: "u" }, { id: "bob" }]);
    storage.fetchIdentityPublicKey.mockImplementation(async (id: string) => ({ publicKey: `pk-${id}` }));
    crypto.wrapChatKey.mockImplementation(async ({ keyId }: { keyId: string }) => ({ keyId, wrapped: true }));
  });

  it("wraps one fresh key for every current member and every new user without opening old versions", async () => {
    const result = await prepareAddMembers("u", "c", ["carol", "dave"]);

    expect(result.wraps.map((item) => item.userId)).toEqual(["u", "bob", "carol", "dave"]);
    expect(crypto.wrapChatKey).toHaveBeenCalledTimes(4);
    const keys = new Set(crypto.wrapChatKey.mock.calls.map(([arg]) => arg.chatKey));
    expect(keys.size).toBe(1);
    expect(crypto.wrapChatKey.mock.calls.every(([arg]) => arg.keyId === result.keyId)).toBe(true);
    expect(storage.fetchIdentityPublicKey.mock.calls.map(([id]) => id)).toEqual(["bob", "carol", "dave"]);
    expect(storage.cacheChatKey).toHaveBeenCalledWith("u", "c", result.keyId, expect.any(Uint8Array));
    expect(crypto.unwrapChatKey).not.toHaveBeenCalled();
    expect(storage.fetchChatKeyWraps).not.toHaveBeenCalled();
    expect(storage.uploadChatKeyVersion).not.toHaveBeenCalled();
  });

  it("fails naming the user when one of several identity keys is missing, caching nothing", async () => {
    storage.fetchIdentityPublicKey.mockImplementation(async (id: string) => {
      if (id === "carol") {
        throw new Error("identity key not found");
      }
      return { publicKey: `pk-${id}` };
    });

    await expect(prepareAddMembers("u", "c", ["carol", "dave"])).rejects.toThrow(
      "identity key not found for carol",
    );
    expect(storage.cacheChatKey).not.toHaveBeenCalled();
  });
});
