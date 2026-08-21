import { hkdf } from "@noble/hashes/hkdf.js";
import { scryptAsync } from "@noble/hashes/scrypt.js";
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

type IdentityBackupKdfParams = {
  n: number;
  r: number;
  p: number;
};

type IdentityBackupEnvelope = {
  alg: "scrypt-aes256gcm";
  kdf: IdentityBackupKdfParams & { salt: string };
  nonce: string;
  ciphertext: string;
};

const IDENTITY_BACKUP_ALG = "scrypt-aes256gcm" as const;
// Tuned for roughly 0.5-1s on typical hardware; scryptAsync yields to the
// event loop between chunks so this does not block the UI thread.
const IDENTITY_BACKUP_SCRYPT_N = 2 ** 17;
const IDENTITY_BACKUP_SCRYPT_R = 8;
const IDENTITY_BACKUP_SCRYPT_P = 1;
const IDENTITY_BACKUP_DK_LEN = 32;
const IDENTITY_BACKUP_GENERIC_ERROR = "wrong passphrase or corrupted file";
const IDENTITY_BACKUP_LEGACY_ERROR = "unsupported or legacy backup format";

async function deriveIdentityBackupKey(
  passphrase: string,
  salt: Uint8Array,
  params: IdentityBackupKdfParams,
): Promise<Uint8Array> {
  return scryptAsync(passphrase, salt, {
    N: params.n,
    r: params.r,
    p: params.p,
    dkLen: IDENTITY_BACKUP_DK_LEN,
  });
}

// Encrypts a StoredIdentity into the passphrase-protected backup envelope
// described in README.md (scrypt-derived AES-256-GCM key, random salt and
// nonce, no format-version field by design). The passphrase is never
// persisted; only the derived key lives in memory for the duration of this
// call.
export async function exportIdentityBackup(identity: StoredIdentity, passphrase: string): Promise<string> {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const kdfParams: IdentityBackupKdfParams = {
    n: IDENTITY_BACKUP_SCRYPT_N,
    r: IDENTITY_BACKUP_SCRYPT_R,
    p: IDENTITY_BACKUP_SCRYPT_P,
  };
  const derivedKey = await deriveIdentityBackupKey(passphrase, salt, kdfParams);
  const aesKey = await importAesKey(derivedKey);
  const nonce = crypto.getRandomValues(new Uint8Array(12));
  const plaintext = utf8Bytes(JSON.stringify(identity));
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: nonce.buffer as ArrayBuffer },
    aesKey,
    plaintext.buffer as ArrayBuffer,
  );

  const envelope: IdentityBackupEnvelope = {
    alg: IDENTITY_BACKUP_ALG,
    kdf: { ...kdfParams, salt: bytesToBase64Url(salt) },
    nonce: bytesToBase64Url(nonce),
    ciphertext: bytesToBase64Url(new Uint8Array(ciphertext)),
  };
  return JSON.stringify(envelope);
}

// Detects the current envelope shape (alg/kdf/nonce/ciphertext, all
// base64url) and rejects anything else — including the legacy plaintext
// {publicKey, secretKey} format this replaces — with a clear error before
// any decrypt attempt is made.
function parseIdentityBackupEnvelope(raw: string): IdentityBackupEnvelope {
  let parsed: Partial<IdentityBackupEnvelope>;
  try {
    parsed = JSON.parse(raw) as Partial<IdentityBackupEnvelope>;
  } catch {
    throw new Error(IDENTITY_BACKUP_LEGACY_ERROR);
  }

  const kdf = parsed.kdf;
  const valid =
    parsed.alg === IDENTITY_BACKUP_ALG &&
    !!kdf &&
    typeof kdf.n === "number" &&
    typeof kdf.r === "number" &&
    typeof kdf.p === "number" &&
    typeof kdf.salt === "string" &&
    isBase64Url(kdf.salt) &&
    typeof parsed.nonce === "string" &&
    isBase64Url(parsed.nonce) &&
    typeof parsed.ciphertext === "string" &&
    isBase64Url(parsed.ciphertext);

  if (!valid) {
    throw new Error(IDENTITY_BACKUP_LEGACY_ERROR);
  }

  return parsed as IdentityBackupEnvelope;
}

// Wrong passphrase and corrupted ciphertext both fail at the AES-GCM auth
// tag check, and both surface the same generic error below — this is
// intentional: a passphrase-specific error message would be an oracle.
export async function importIdentityBackup(raw: string, passphrase: string): Promise<StoredIdentity> {
  const envelope = parseIdentityBackupEnvelope(raw);
  const salt = base64UrlToBytes(envelope.kdf.salt);
  const derivedKey = await deriveIdentityBackupKey(passphrase, salt, envelope.kdf);
  const aesKey = await importAesKey(derivedKey);

  let identity: Partial<StoredIdentity>;
  try {
    const plaintext = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: base64UrlToBytes(envelope.nonce).buffer as ArrayBuffer },
      aesKey,
      base64UrlToBytes(envelope.ciphertext).buffer as ArrayBuffer,
    );
    identity = JSON.parse(new TextDecoder().decode(plaintext)) as Partial<StoredIdentity>;
  } catch {
    throw new Error(IDENTITY_BACKUP_GENERIC_ERROR);
  }

  if (typeof identity.publicKey !== "string" || typeof identity.secretKey !== "string") {
    throw new Error(IDENTITY_BACKUP_GENERIC_ERROR);
  }

  return { publicKey: identity.publicKey, secretKey: identity.secretKey };
}
