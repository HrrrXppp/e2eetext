import "fake-indexeddb/auto";
import { beforeEach, describe, expect, it } from "vitest";
import {
  clearAllKeys,
  loadChatPrivateKey,
  loadOwnKeyPair,
  saveChatPrivateKey,
  saveOwnKeyPair,
} from "./keyStore";
import { exportOwnKeyPairEncrypted, importOwnKeyPairFromBackup } from "./keyBackup";
import { generateKemKeyPair } from "./crypto";

beforeEach(() => {
  indexedDB = new IDBFactory();
});

// fake-indexeddb's structured clone produces Uint8Array instances from a
// different realm than the ones created in this test file, so `toEqual`
// (which checks prototype identity for typed arrays) reports a false
// mismatch even when the bytes are identical. Compare by value instead.
function bytesEqual(a: Uint8Array, b: Uint8Array): boolean {
  return a.length === b.length && Array.from(a).every((byte, i) => byte === b[i]);
}

describe("saveOwnKeyPair / loadOwnKeyPair", () => {
  it("returns null when nothing is stored", async () => {
    expect(await loadOwnKeyPair("user-1")).toBeNull();
  });

  it("round trips a keypair", async () => {
    const keyPair = generateKemKeyPair();
    await saveOwnKeyPair("user-1", keyPair);

    const loaded = await loadOwnKeyPair("user-1");
    expect(loaded).not.toBeNull();
    expect(bytesEqual(loaded!.publicKey, keyPair.publicKey)).toBe(true);
    expect(bytesEqual(loaded!.secretKey, keyPair.secretKey)).toBe(true);
  });

  it("keeps keypairs for different users separate", async () => {
    const a = generateKemKeyPair();
    const b = generateKemKeyPair();
    await saveOwnKeyPair("user-1", a);
    await saveOwnKeyPair("user-2", b);

    expect(bytesEqual((await loadOwnKeyPair("user-1"))!.secretKey, a.secretKey)).toBe(true);
    expect(bytesEqual((await loadOwnKeyPair("user-2"))!.secretKey, b.secretKey)).toBe(true);
  });
});

describe("saveChatPrivateKey / loadChatPrivateKey", () => {
  it("returns null when nothing is stored", async () => {
    expect(await loadChatPrivateKey("user-1", "chat-1")).toBeNull();
  });

  it("round trips a chat private key", async () => {
    const chatPrivateKey = crypto.getRandomValues(new Uint8Array(3168));
    await saveChatPrivateKey("user-1", "chat-1", chatPrivateKey);

    const loaded = await loadChatPrivateKey("user-1", "chat-1");
    expect(bytesEqual(loaded!, chatPrivateKey)).toBe(true);
  });

  it("keeps keys for different chats separate", async () => {
    const key1 = crypto.getRandomValues(new Uint8Array(3168));
    const key2 = crypto.getRandomValues(new Uint8Array(3168));
    await saveChatPrivateKey("user-1", "chat-1", key1);
    await saveChatPrivateKey("user-1", "chat-2", key2);

    expect(bytesEqual((await loadChatPrivateKey("user-1", "chat-1"))!, key1)).toBe(true);
    expect(bytesEqual((await loadChatPrivateKey("user-1", "chat-2"))!, key2)).toBe(true);
  });
});

describe("clearAllKeys", () => {
  it("removes the own keypair and all chat keys for that user, leaving other users intact", async () => {
    const ownKeyPair = generateKemKeyPair();
    const otherOwnKeyPair = generateKemKeyPair();
    await saveOwnKeyPair("user-1", ownKeyPair);
    await saveOwnKeyPair("user-2", otherOwnKeyPair);
    await saveChatPrivateKey("user-1", "chat-1", crypto.getRandomValues(new Uint8Array(3168)));
    await saveChatPrivateKey("user-1", "chat-2", crypto.getRandomValues(new Uint8Array(3168)));
    const otherUserChatKey = crypto.getRandomValues(new Uint8Array(3168));
    await saveChatPrivateKey("user-2", "chat-1", otherUserChatKey);

    await clearAllKeys("user-1");

    expect(await loadOwnKeyPair("user-1")).toBeNull();
    expect(await loadChatPrivateKey("user-1", "chat-1")).toBeNull();
    expect(await loadChatPrivateKey("user-1", "chat-2")).toBeNull();

    expect(await loadOwnKeyPair("user-2")).not.toBeNull();
    expect(bytesEqual((await loadChatPrivateKey("user-2", "chat-1"))!, otherUserChatKey)).toBe(true);
  });
});

describe("key backup export/import", () => {
  it("exports and re-imports a keypair with the correct passphrase", async () => {
    const keyPair = generateKemKeyPair();
    await saveOwnKeyPair("user-1", keyPair);

    const blob = await exportOwnKeyPairEncrypted("user-1", "correct horse battery staple");
    const restored = await importOwnKeyPairFromBackup(blob, "correct horse battery staple");

    expect(restored.publicKey).toEqual(keyPair.publicKey);
    expect(restored.secretKey).toEqual(keyPair.secretKey);
  });

  it("throws on the wrong passphrase", async () => {
    const keyPair = generateKemKeyPair();
    await saveOwnKeyPair("user-1", keyPair);

    const blob = await exportOwnKeyPairEncrypted("user-1", "correct horse battery staple");

    await expect(importOwnKeyPairFromBackup(blob, "wrong passphrase")).rejects.toThrow();
  });

  it("throws on a malformed backup file", async () => {
    const blob = new Blob(["not json"], { type: "application/json" });
    await expect(importOwnKeyPairFromBackup(blob, "whatever")).rejects.toThrow();
  });

  it("throws when there is no keypair to export", async () => {
    await expect(exportOwnKeyPairEncrypted("no-such-user", "passphrase")).rejects.toThrow();
  });
});
