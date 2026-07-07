import { describe, expect, it } from "vitest";
import {
  decryptMessage,
  encryptMessage,
  generateKemKeyPair,
  unwrapChatPrivateKey,
  wrapChatPrivateKeyForMember,
} from "./crypto";

describe("generateKemKeyPair", () => {
  it("produces ML-KEM-1024 sized keys", () => {
    const { publicKey, secretKey } = generateKemKeyPair();
    expect(publicKey.length).toBe(1568);
    expect(secretKey.length).toBe(3168);
  });

  it("produces different keys on each call", () => {
    const a = generateKemKeyPair();
    const b = generateKemKeyPair();
    expect(a.secretKey).not.toEqual(b.secretKey);
  });
});

describe("wrapChatPrivateKeyForMember / unwrapChatPrivateKey", () => {
  it("round trips the chat private key", async () => {
    const member = generateKemKeyPair();
    const chatPrivateKey = crypto.getRandomValues(new Uint8Array(3168));

    const { kemCiphertext, wrappedChatPrivateKey } = await wrapChatPrivateKeyForMember(
      chatPrivateKey,
      member.publicKey,
    );
    expect(kemCiphertext.length).toBe(1568);

    const recovered = await unwrapChatPrivateKey(kemCiphertext, wrappedChatPrivateKey, member.secretKey);
    expect(recovered).toEqual(chatPrivateKey);
  });

  it("fails to unwrap with the wrong secret key", async () => {
    const member = generateKemKeyPair();
    const attacker = generateKemKeyPair();
    const chatPrivateKey = crypto.getRandomValues(new Uint8Array(3168));

    const { kemCiphertext, wrappedChatPrivateKey } = await wrapChatPrivateKeyForMember(
      chatPrivateKey,
      member.publicKey,
    );

    await expect(unwrapChatPrivateKey(kemCiphertext, wrappedChatPrivateKey, attacker.secretKey)).rejects.toThrow();
  });

  it("fails to unwrap when the ciphertext is tampered with", async () => {
    const member = generateKemKeyPair();
    const chatPrivateKey = crypto.getRandomValues(new Uint8Array(3168));

    const { kemCiphertext, wrappedChatPrivateKey } = await wrapChatPrivateKeyForMember(
      chatPrivateKey,
      member.publicKey,
    );
    const tampered = new Uint8Array(wrappedChatPrivateKey);
    tampered[0] ^= 0xff;

    await expect(unwrapChatPrivateKey(kemCiphertext, tampered, member.secretKey)).rejects.toThrow();
  });
});

describe("encryptMessage / decryptMessage", () => {
  it("round trips a plain message", async () => {
    const chat = generateKemKeyPair();
    const { kemCiphertext, data } = await encryptMessage(chat.publicKey, "hello world");
    const plaintext = await decryptMessage(kemCiphertext, data, chat.secretKey);
    expect(plaintext).toBe("hello world");
  });

  it("round trips an empty message", async () => {
    const chat = generateKemKeyPair();
    const { kemCiphertext, data } = await encryptMessage(chat.publicKey, "");
    const plaintext = await decryptMessage(kemCiphertext, data, chat.secretKey);
    expect(plaintext).toBe("");
  });

  it("round trips unicode/emoji content", async () => {
    const chat = generateKemKeyPair();
    const message = "Привет 👋 世界";
    const { kemCiphertext, data } = await encryptMessage(chat.publicKey, message);
    const plaintext = await decryptMessage(kemCiphertext, data, chat.secretKey);
    expect(plaintext).toBe(message);
  });

  it("round trips a large message", async () => {
    const chat = generateKemKeyPair();
    const message = "x".repeat(50_000);
    const { kemCiphertext, data } = await encryptMessage(chat.publicKey, message);
    const plaintext = await decryptMessage(kemCiphertext, data, chat.secretKey);
    expect(plaintext).toBe(message);
  });

  it("produces different ciphertext for the same message each time", async () => {
    const chat = generateKemKeyPair();
    const a = await encryptMessage(chat.publicKey, "hello");
    const b = await encryptMessage(chat.publicKey, "hello");
    expect(a.data).not.toEqual(b.data);
    expect(a.kemCiphertext).not.toEqual(b.kemCiphertext);
  });

  it("fails to decrypt when the ciphertext is tampered with", async () => {
    const chat = generateKemKeyPair();
    const { kemCiphertext, data } = await encryptMessage(chat.publicKey, "hello");
    const tampered = new Uint8Array(data);
    tampered[tampered.length - 1] ^= 0xff;

    await expect(decryptMessage(kemCiphertext, tampered, chat.secretKey)).rejects.toThrow();
  });

  it("fails to decrypt with the wrong chat private key", async () => {
    const chat = generateKemKeyPair();
    const other = generateKemKeyPair();
    const { kemCiphertext, data } = await encryptMessage(chat.publicKey, "hello");

    await expect(decryptMessage(kemCiphertext, data, other.secretKey)).rejects.toThrow();
  });
});
