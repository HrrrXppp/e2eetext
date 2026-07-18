import { hkdf } from "@noble/hashes/hkdf.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { ml_kem768_x25519 } from "@noble/post-quantum/hybrid.js";
import { randomBytes } from "@noble/post-quantum/utils.js";
import { base64UrlToBytes, bytesToBase64Url, isBase64Url, utf8Bytes } from "@/lib/e2ee/encoding";
import {
  E2EE_IDENTITY_ALG,
  E2EE_MESSAGE_ALG,
  E2EE_WRAP_ALG,
  MAX_MESSAGE_PLAINTEXT_BYTES,
  type ChatKeyWrap,
  type IdentityPublicKey,
  type MessageEnvelope,
  type StoredIdentity,
} from "@/lib/e2ee/types";

const hybridKem = ml_kem768_x25519;

async function importAesKey(rawKey: Uint8Array): Promise<CryptoKey> {
  return crypto.subtle.importKey("raw", rawKey.buffer.slice(rawKey.byteOffset, rawKey.byteOffset + rawKey.byteLength) as ArrayBuffer, "AES-GCM", false, [
    "encrypt",
    "decrypt",
  ]);
}

function deriveWrapKey(sharedSecret: Uint8Array, keyId: string): Uint8Array {
  return hkdf(sha256, sharedSecret, undefined, utf8Bytes(`e2ee_v1:chat-key-wrap:${keyId}`), 32);
}

function deriveMessageKey(chatKey: Uint8Array, keyId: string): Uint8Array {
  return hkdf(sha256, chatKey, undefined, utf8Bytes(`e2ee_v1:chat-key:${keyId}`), 32);
}

export function generateIdentityKeyPair(): StoredIdentity {
  const seed = randomBytes(hybridKem.lengths.seed);
  const { publicKey, secretKey } = hybridKem.keygen(seed);
  return {
    publicKey: bytesToBase64Url(publicKey),
    secretKey: bytesToBase64Url(secretKey),
  };
}

export function toIdentityPublicKey(identity: StoredIdentity): IdentityPublicKey {
  return {
    v: 1,
    alg: E2EE_IDENTITY_ALG,
    publicKey: identity.publicKey,
  };
}

export function parseIdentityPublicKey(value: unknown): IdentityPublicKey {
  if (!value || typeof value !== "object") {
    throw new Error("invalid identity public key");
  }
  const key = value as Partial<IdentityPublicKey>;
  if (key.v !== 1 || key.alg !== E2EE_IDENTITY_ALG || !key.publicKey || !isBase64Url(key.publicKey)) {
    throw new Error("unsupported identity public key");
  }
  return key as IdentityPublicKey;
}

export function generateChatKey(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(32));
}

export async function wrapChatKey(input: {
  chatKey: Uint8Array;
  keyId: string;
  recipientPublicKey: IdentityPublicKey;
}): Promise<ChatKeyWrap> {
  const recipientPk = base64UrlToBytes(input.recipientPublicKey.publicKey);
  const { cipherText, sharedSecret } = hybridKem.encapsulate(recipientPk);
  const wrapKey = deriveWrapKey(sharedSecret, input.keyId);
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const aesKey = await importAesKey(wrapKey);
  const ciphertext = await crypto.subtle.encrypt({ name: "AES-GCM", iv: nonce.buffer as ArrayBuffer }, aesKey, input.chatKey.buffer as ArrayBuffer);

  return {
    v: 1,
    alg: E2EE_WRAP_ALG,
    keyId: input.keyId,
    kemCiphertext: bytesToBase64Url(cipherText),
    nonce: bytesToBase64Url(nonce),
    ciphertext: bytesToBase64Url(new Uint8Array(ciphertext)),
  };
}

export async function unwrapChatKey(input: {
  wrap: ChatKeyWrap;
  identity: StoredIdentity;
}): Promise<Uint8Array> {
  if (input.wrap.v !== 1 || input.wrap.alg !== E2EE_WRAP_ALG) {
    throw new Error("unsupported chat key wrap");
  }

  const sharedSecret = hybridKem.decapsulate(base64UrlToBytes(input.wrap.kemCiphertext), base64UrlToBytes(input.identity.secretKey));
  const wrapKey = deriveWrapKey(sharedSecret, input.wrap.keyId);
  const aesKey = await importAesKey(wrapKey);
  const chatKey = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: base64UrlToBytes(input.wrap.nonce).buffer as ArrayBuffer },
    aesKey,
    base64UrlToBytes(input.wrap.ciphertext).buffer as ArrayBuffer,
  );

  return new Uint8Array(chatKey);
}

export async function encryptMessage(input: {
  plaintext: string;
  keyId: string;
  chatKey: Uint8Array;
}): Promise<MessageEnvelope> {
  const plaintextBytes = utf8Bytes(input.plaintext);
  if (plaintextBytes.byteLength > MAX_MESSAGE_PLAINTEXT_BYTES) {
    throw new Error("message exceeds 100 KiB limit");
  }

  const messageKey = deriveMessageKey(input.chatKey, input.keyId);
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const aesKey = await importAesKey(messageKey);
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: nonce.buffer as ArrayBuffer },
    aesKey,
    plaintextBytes.buffer as ArrayBuffer,
  );

  return {
    v: 1,
    alg: E2EE_MESSAGE_ALG,
    keyId: input.keyId,
    nonce: bytesToBase64Url(nonce),
    ciphertext: bytesToBase64Url(new Uint8Array(ciphertext)),
  };
}

export async function decryptMessage(input: {
  envelope: MessageEnvelope;
  chatKey: Uint8Array;
}): Promise<string> {
  if (input.envelope.v !== 1 || input.envelope.alg !== E2EE_MESSAGE_ALG) {
    throw new Error("unsupported message envelope");
  }

  const messageKey = deriveMessageKey(input.chatKey, input.envelope.keyId);
  const aesKey = await importAesKey(messageKey);
  const plaintext = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: base64UrlToBytes(input.envelope.nonce).buffer as ArrayBuffer,
    },
    aesKey,
    base64UrlToBytes(input.envelope.ciphertext).buffer as ArrayBuffer,
  );

  return new TextDecoder().decode(plaintext);
}

export function parseMessageEnvelope(raw: string): MessageEnvelope | null {
  try {
    const parsed = JSON.parse(raw) as Partial<MessageEnvelope>;
    if (
      parsed.v !== 1 ||
      parsed.alg !== E2EE_MESSAGE_ALG ||
      typeof parsed.keyId !== "string" ||
      typeof parsed.nonce !== "string" ||
      typeof parsed.ciphertext !== "string"
    ) {
      return null;
    }
    return parsed as MessageEnvelope;
  } catch {
    return null;
  }
}

export function serializeMessageEnvelope(envelope: MessageEnvelope): string {
  return JSON.stringify(envelope);
}

export function newKeyId(): string {
  return crypto.randomUUID();
}

export function exportIdentityBackup(identity: StoredIdentity): string {
  return JSON.stringify(identity);
}

export function importIdentityBackup(raw: string): StoredIdentity {
  const parsed = JSON.parse(raw) as Partial<StoredIdentity>;
  if (typeof parsed.publicKey !== "string" || typeof parsed.secretKey !== "string") {
    throw new Error("invalid identity backup");
  }
  return { publicKey: parsed.publicKey, secretKey: parsed.secretKey };
}
