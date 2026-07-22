import type { StoredIdentity } from "@/lib/e2ee/types";
import { clearLocalCryptoKey, decryptLocalValue, encryptLocalValue } from "@/lib/e2ee/localCrypto";
import {
  exportIdentityBackup,
  importIdentityBackup,
  parseIdentityPublicKey,
  toIdentityPublicKey,
} from "@/lib/e2ee/crypto";
import { base64UrlToBytes, bytesToBase64Url } from "@/lib/e2ee/encoding";
import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";
import type { ChatKeyWrapResponse, IdentityPublicKey } from "@/lib/e2ee/types";

const IDENTITY_STORAGE_PREFIX = "e2ee_identity_v1";

function identityStorageKey(userId: string): string {
  return `${IDENTITY_STORAGE_PREFIX}:${userId}`;
}

function parseStoredIdentity(raw: string): StoredIdentity | null {
  try {
    const parsed = JSON.parse(raw) as Partial<StoredIdentity>;
    if (typeof parsed.publicKey !== "string" || typeof parsed.secretKey !== "string") {
      return null;
    }
    return { publicKey: parsed.publicKey, secretKey: parsed.secretKey };
  } catch {
    return null;
  }
}

// Matches the {v, iv, ciphertext} shape encryptLocalValue produces (see
// localCrypto.ts). Checked up front so a corrupt/unrelated localStorage
// value that is neither a valid plaintext identity nor a valid envelope
// fails closed immediately, instead of being handed to decryptLocalValue
// only to throw there.
function looksLikeEncryptedEnvelope(raw: string): boolean {
  try {
    const parsed = JSON.parse(raw) as { v?: number; iv?: string; ciphertext?: string };
    return parsed.v === 1 && typeof parsed.iv === "string" && typeof parsed.ciphertext === "string";
  } catch {
    return false;
  }
}

// Both legacy plaintext identities and the encrypted envelope (from
// encryptLocalValue) serialize as JSON objects, so a "starts with {" prefix
// check can't tell them apart — every encrypted value would be misread as
// plaintext and fail parseStoredIdentity's field check (missing
// publicKey/secretKey, since it actually has v/iv/ciphertext), silently
// skipping decryption entirely. Discriminate by shape instead: only treat it
// as legacy plaintext if it actually parses as one.
async function readIdentityPayload(
  userId: string,
  raw: string,
): Promise<{ identity: StoredIdentity; wasPlaintext: boolean } | null> {
  const plaintext = parseStoredIdentity(raw);
  if (plaintext) {
    return { identity: plaintext, wasPlaintext: true };
  }

  if (!looksLikeEncryptedEnvelope(raw)) {
    return null;
  }

  try {
    const decrypted = await decryptLocalValue(userId, raw);
    const identity = parseStoredIdentity(decrypted);
    return identity ? { identity, wasPlaintext: false } : null;
  } catch {
    return null;
  }
}

export async function loadStoredIdentity(userId: string): Promise<StoredIdentity | null> {
  const scopedRaw = localStorage.getItem(identityStorageKey(userId));
  if (!scopedRaw) {
    return null;
  }

  const result = await readIdentityPayload(userId, scopedRaw);
  if (!result) {
    return null;
  }

  if (result.wasPlaintext) {
    await saveStoredIdentity(userId, result.identity);
  }
  return result.identity;
}

export async function saveStoredIdentity(userId: string, identity: StoredIdentity): Promise<void> {
  const encrypted = await encryptLocalValue(userId, JSON.stringify(identity));
  localStorage.setItem(identityStorageKey(userId), encrypted);
}

export async function exportStoredIdentityBackup(userId: string): Promise<string> {
  const identity = await loadStoredIdentity(userId);
  if (!identity) {
    throw new Error("identity not found");
  }
  return exportIdentityBackup(identity);
}

export async function importStoredIdentityBackup(userId: string, raw: string): Promise<StoredIdentity> {
  const identity = importIdentityBackup(raw);
  await saveStoredIdentity(userId, identity);
  return identity;
}

// fetch() has no default timeout — an unresponsive server would otherwise
// hang this indefinitely. Callers now treat this as background work (see
// ensureIdentityKeys/prepareE2EEChatCreation in session.ts), but it should
// still fail on its own rather than hold a dangling promise/connection open.
const UPLOAD_IDENTITY_KEY_TIMEOUT_MS = 15_000;

export async function uploadIdentityKey(userId: string, identity: StoredIdentity): Promise<void> {
  const response = await fetch(`${API_V1}/user/${userId}/identity-key`, {
    method: "PUT",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ publicKey: toIdentityPublicKey(identity) }),
    signal: AbortSignal.timeout(UPLOAD_IDENTITY_KEY_TIMEOUT_MS),
  });

  if (!response.ok) {
    throw new Error("failed to upload identity key");
  }
}

export async function fetchIdentityPublicKey(userId: string): Promise<IdentityPublicKey> {
  const response = await fetch(`${API_V1}/user/${userId}/identity-key`, {
    headers: authHeaders(),
  });

  if (!response.ok) {
    throw new Error("identity key not found");
  }

  const data = (await response.json()) as { publicKey?: unknown };
  return parseIdentityPublicKey(data.publicKey);
}

export async function fetchChatKeyWraps(chatId: string): Promise<ChatKeyWrapResponse[]> {
  const response = await fetch(`${API_V1}/chat/${chatId}/key-wraps`, {
    headers: authHeaders(),
  });

  if (!response.ok) {
    throw new Error("failed to fetch chat key wraps");
  }

  const data = (await response.json()) as { wraps?: ChatKeyWrapResponse[] };
  return data.wraps ?? [];
}

const chatKeyCache = new Map<string, string>();

export async function cacheChatKey(
  userId: string,
  chatId: string,
  keyId: string,
  chatKey: Uint8Array,
): Promise<void> {
  chatKeyCache.set(`${userId}:${chatId}:${keyId}`, await encryptLocalValue(userId, bytesToBase64Url(chatKey)));
}

export async function getCachedChatKey(
  userId: string,
  chatId: string,
  keyId: string,
): Promise<Uint8Array | null> {
  const encrypted = chatKeyCache.get(`${userId}:${chatId}:${keyId}`);
  if (!encrypted) {
    return null;
  }

  const raw = await decryptLocalValue(userId, encrypted);
  return base64UrlToBytes(raw);
}

export function clearChatKeyCache(userId?: string): void {
  if (!userId) {
    chatKeyCache.clear();
    return;
  }

  const prefix = `${userId}:`;
  for (const key of [...chatKeyCache.keys()]) {
    if (key.startsWith(prefix)) {
      chatKeyCache.delete(key);
    }
  }
}

export async function clearE2EEState(userId: string): Promise<void> {
  localStorage.removeItem(identityStorageKey(userId));
  await clearLocalCryptoKey(userId);
  clearChatKeyCache(userId);
}
