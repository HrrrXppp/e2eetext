import { describe, expect, it } from "vitest";
import {
  decryptMessage,
  encryptMessage,
  exportIdentityBackup,
  generateChatKey,
  generateIdentityKeyPair,
  importIdentityBackup,
  newKeyId,
  parseMessageEnvelope,
  serializeMessageEnvelope,
  toIdentityPublicKey,
  unwrapChatKey,
  wrapChatKey,
} from "@/lib/e2ee/crypto";

describe("e2ee crypto", () => {
  it("wraps and unwraps chat keys", async () => {
    const identity = generateIdentityKeyPair();
    const keyId = newKeyId();
    const chatKey = generateChatKey();

    const wrap = await wrapChatKey({
      chatKey,
      keyId,
      recipientPublicKey: toIdentityPublicKey(identity),
    });

    const unwrapped = await unwrapChatKey({
      wrap,
      identity,
    });

    expect(unwrapped).toEqual(chatKey);
  });

  it("encrypts and decrypts messages", async () => {
    const keyId = newKeyId();
    const chatKey = generateChatKey();

    const envelope = await encryptMessage({
      plaintext: "hello e2ee",
      keyId,
      chatKey,
    });

    const serialized = serializeMessageEnvelope(envelope);
    const parsed = parseMessageEnvelope(serialized);
    expect(parsed).toEqual(envelope);

    const plaintext = await decryptMessage({
      envelope,
      chatKey,
    });

    expect(plaintext).toBe("hello e2ee");
  });
});

describe("identity backup export/import", () => {
  it("round-trips an identity through export and import with the same passphrase", async () => {
    const identity = generateIdentityKeyPair();
    const backup = await exportIdentityBackup(identity, "correct horse battery staple");

    const restored = await importIdentityBackup(backup, "correct horse battery staple");

    expect(restored).toEqual(identity);
  });

  it("accepts a passphrase with no minimum length", async () => {
    const identity = generateIdentityKeyPair();
    const backup = await exportIdentityBackup(identity, "x");

    const restored = await importIdentityBackup(backup, "x");

    expect(restored).toEqual(identity);
  });

  it("fails with a generic error on the wrong passphrase", async () => {
    const identity = generateIdentityKeyPair();
    const backup = await exportIdentityBackup(identity, "correct horse battery staple");

    await expect(importIdentityBackup(backup, "wrong passphrase")).rejects.toThrow(
      "wrong passphrase or corrupted file",
    );
  });

  it("rejects a legacy plaintext backup with a clear format error", async () => {
    const identity = generateIdentityKeyPair();
    const legacyPlaintext = JSON.stringify(identity);

    await expect(importIdentityBackup(legacyPlaintext, "any passphrase")).rejects.toThrow(
      "unsupported or legacy backup format",
    );
  });

  it("rejects corrupted ciphertext with the same generic error as a wrong passphrase", async () => {
    const identity = generateIdentityKeyPair();
    const backup = await exportIdentityBackup(identity, "correct horse battery staple");
    const envelope = JSON.parse(backup) as { ciphertext: string };
    envelope.ciphertext = `${envelope.ciphertext.slice(0, -4)}abcd`;

    await expect(
      importIdentityBackup(JSON.stringify(envelope), "correct horse battery staple"),
    ).rejects.toThrow("wrong passphrase or corrupted file");
  });
});
