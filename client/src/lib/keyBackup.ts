import type { KemKeyPair } from "./crypto";
import { decodeBase64, encodeBase64 } from "./bytes";
import { loadOwnKeyPair } from "./keyStore";

const PBKDF2_ITERATIONS = 600_000;
const SALT_LENGTH = 16;
const NONCE_LENGTH = 12;
const BACKUP_VERSION = 1;

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

type BackupFile = {
  version: number;
  salt: string;
  iterations: number;
  nonce: string;
  ciphertext: string;
};

async function deriveKeyFromPassphrase(passphrase: string, salt: Uint8Array, iterations: number): Promise<CryptoKey> {
  const baseKey = await crypto.subtle.importKey(
    "raw",
    textEncoder.encode(passphrase) as BufferSource,
    "PBKDF2",
    false,
    ["deriveKey"],
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: salt as BufferSource, iterations, hash: "SHA-256" },
    baseKey,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}

export async function exportOwnKeyPairEncrypted(userId: string, passphrase: string): Promise<Blob> {
  const keyPair = await loadOwnKeyPair(userId);
  if (!keyPair) {
    throw new Error(`no local keypair found for user ${userId}`);
  }

  const salt = crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
  const nonce = crypto.getRandomValues(new Uint8Array(NONCE_LENGTH));
  const key = await deriveKeyFromPassphrase(passphrase, salt, PBKDF2_ITERATIONS);

  const plaintext = textEncoder.encode(
    JSON.stringify({
      publicKey: encodeBase64(keyPair.publicKey),
      secretKey: encodeBase64(keyPair.secretKey),
    }),
  );
  const ciphertext = new Uint8Array(
    await crypto.subtle.encrypt({ name: "AES-GCM", iv: nonce as BufferSource }, key, plaintext as BufferSource),
  );

  const backup: BackupFile = {
    version: BACKUP_VERSION,
    salt: encodeBase64(salt),
    iterations: PBKDF2_ITERATIONS,
    nonce: encodeBase64(nonce),
    ciphertext: encodeBase64(ciphertext),
  };

  return new Blob([JSON.stringify(backup)], { type: "application/json" });
}

function readBlobAsText(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(reader.error ?? new Error("failed to read backup file"));
    reader.readAsText(blob);
  });
}

export async function importOwnKeyPairFromBackup(file: Blob, passphrase: string): Promise<KemKeyPair> {
  const text = await readBlobAsText(file);

  let backup: BackupFile;
  try {
    backup = JSON.parse(text);
  } catch {
    throw new Error("backup file is not valid JSON");
  }
  if (
    typeof backup.salt !== "string" ||
    typeof backup.nonce !== "string" ||
    typeof backup.ciphertext !== "string" ||
    typeof backup.iterations !== "number"
  ) {
    throw new Error("backup file is missing required fields");
  }

  const salt = decodeBase64(backup.salt);
  const nonce = decodeBase64(backup.nonce);
  const ciphertext = decodeBase64(backup.ciphertext);
  const key = await deriveKeyFromPassphrase(passphrase, salt, backup.iterations);

  let plaintext: ArrayBuffer;
  try {
    plaintext = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: nonce as BufferSource },
      key,
      ciphertext as BufferSource,
    );
  } catch {
    throw new Error("wrong passphrase or corrupted backup file");
  }

  const parsed = JSON.parse(textDecoder.decode(plaintext));
  return {
    publicKey: decodeBase64(parsed.publicKey),
    secretKey: decodeBase64(parsed.secretKey),
  };
}
