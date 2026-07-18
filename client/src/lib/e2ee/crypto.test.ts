import { describe, expect, it } from "vitest";
import {
  decryptMessage,
  encryptMessage,
  generateChatKey,
  generateIdentityKeyPair,
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
