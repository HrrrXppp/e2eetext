import { ml_kem1024 } from "@noble/post-quantum/ml-kem.js";

export type KemKeyPair = { publicKey: Uint8Array; secretKey: Uint8Array };

const HKDF_INFO_MESSAGE_KEY = "e2eetext/message-key/v1";
const HKDF_INFO_CHAT_KEY_WRAP = "e2eetext/chat-key-wrap/v1";
const GCM_NONCE_LENGTH = 12;

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

export function generateKemKeyPair(): KemKeyPair {
  const { publicKey, secretKey } = ml_kem1024.keygen();
  return { publicKey, secretKey };
}

async function deriveAesGcmKey(sharedSecret: Uint8Array, info: string): Promise<CryptoKey> {
  const baseKey = await crypto.subtle.importKey(
    "raw",
    sharedSecret as BufferSource,
    "HKDF",
    false,
    ["deriveKey"],
  );
  return crypto.subtle.deriveKey(
    {
      name: "HKDF",
      hash: "SHA-384",
      salt: new Uint8Array(0),
      info: textEncoder.encode(info),
    },
    baseKey,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

async function aesGcmSeal(key: CryptoKey, plaintext: Uint8Array): Promise<Uint8Array> {
  const nonce = crypto.getRandomValues(new Uint8Array(GCM_NONCE_LENGTH));
  const ciphertext = new Uint8Array(
    await crypto.subtle.encrypt({ name: "AES-GCM", iv: nonce as BufferSource }, key, plaintext as BufferSource),
  );
  const out = new Uint8Array(nonce.length + ciphertext.length);
  out.set(nonce, 0);
  out.set(ciphertext, nonce.length);
  return out;
}

async function aesGcmOpen(key: CryptoKey, sealed: Uint8Array): Promise<Uint8Array> {
  const nonce = sealed.subarray(0, GCM_NONCE_LENGTH);
  const ciphertext = sealed.subarray(GCM_NONCE_LENGTH);
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: nonce as BufferSource },
    key,
    ciphertext as BufferSource,
  );
  return new Uint8Array(plaintext);
}

export async function wrapChatPrivateKeyForMember(
  chatPrivateKey: Uint8Array,
  memberPublicKey: Uint8Array,
): Promise<{ kemCiphertext: Uint8Array; wrappedChatPrivateKey: Uint8Array }> {
  const { cipherText, sharedSecret } = ml_kem1024.encapsulate(memberPublicKey);
  const key = await deriveAesGcmKey(sharedSecret, HKDF_INFO_CHAT_KEY_WRAP);
  const wrappedChatPrivateKey = await aesGcmSeal(key, chatPrivateKey);
  return { kemCiphertext: cipherText, wrappedChatPrivateKey };
}

export async function unwrapChatPrivateKey(
  kemCiphertext: Uint8Array,
  wrappedChatPrivateKey: Uint8Array,
  ownSecretKey: Uint8Array,
): Promise<Uint8Array> {
  const sharedSecret = ml_kem1024.decapsulate(kemCiphertext, ownSecretKey);
  const key = await deriveAesGcmKey(sharedSecret, HKDF_INFO_CHAT_KEY_WRAP);
  return aesGcmOpen(key, wrappedChatPrivateKey);
}

export async function encryptMessage(
  chatPublicKey: Uint8Array,
  plaintext: string,
): Promise<{ kemCiphertext: Uint8Array; data: Uint8Array }> {
  const { cipherText, sharedSecret } = ml_kem1024.encapsulate(chatPublicKey);
  const key = await deriveAesGcmKey(sharedSecret, HKDF_INFO_MESSAGE_KEY);
  const data = await aesGcmSeal(key, textEncoder.encode(plaintext));
  return { kemCiphertext: cipherText, data };
}

export async function decryptMessage(
  kemCiphertext: Uint8Array,
  data: Uint8Array,
  chatPrivateKey: Uint8Array,
): Promise<string> {
  const sharedSecret = ml_kem1024.decapsulate(kemCiphertext, chatPrivateKey);
  const key = await deriveAesGcmKey(sharedSecret, HKDF_INFO_MESSAGE_KEY);
  const plaintext = await aesGcmOpen(key, data);
  return textDecoder.decode(plaintext);
}
