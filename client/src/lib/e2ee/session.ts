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
import { getChatMembers } from "@/lib/chats";
import {
  cacheChatKey,
  fetchChatKeyWraps,
  fetchIdentityPublicKey,
  getCachedChatKey,
  loadStoredIdentity,
  saveStoredIdentity,
  uploadChatKeyVersion,
  uploadIdentityKey,
} from "@/lib/e2ee/storage";
import type { ChatKeyWrap, IdentityPublicKey, StoredIdentity } from "@/lib/e2ee/types";
import type { Message } from "@/lib/messages";

export type EnsureIdentityKeysResult = {
  identity: StoredIdentity;
  created: boolean;
};

export async function ensureIdentityKeys(userId: string): Promise<EnsureIdentityKeysResult> {
  let identity = await loadStoredIdentity(userId);
  let created = false;
  if (!identity) {
    identity = generateIdentityKeyPair();
    created = true;
    await saveStoredIdentity(userId, identity);
  }

  // Not awaited: the current session never needs its own uploaded key back
  // (prepareE2EEChatCreation below wraps for itself from the in-memory
  // identity, never a server fetch) — uploading only makes this user's key
  // discoverable to *other* users. Blocking sign-in on this network round
  // trip was making the whole UI hang on any upload slowness with no
  // timeout of its own, which showed up as an ~80s stall on a single click
  // in CI (traced to this fetch). A failed/slow upload is retried the next
  // time this runs (next sign-in, or the fallback in
  // prepareE2EEChatCreation).
  uploadIdentityKey(userId, identity).catch((error) => {
    console.error("uploadIdentityKey failed", error);
  });
  return { identity, created };
}

export async function prepareE2EEChatCreation(input: {
  memberUserIds: string[];
  currentUserId: string;
}): Promise<{ keyId: string; chatKey: Uint8Array; wraps: { userId: string; wrap: ChatKeyWrap }[] }> {
  let identity = await loadStoredIdentity(input.currentUserId);
  if (!identity) {
    identity = generateIdentityKeyPair();
    await saveStoredIdentity(input.currentUserId, identity);
    // Not awaited — see the matching comment in ensureIdentityKeys above.
    uploadIdentityKey(input.currentUserId, identity).catch((error) => {
      console.error("uploadIdentityKey failed", error);
    });
  }

  return createWrappedChatKey(input.memberUserIds, input.currentUserId, identity);
}

async function createWrappedChatKey(
  memberUserIds: string[],
  currentUserId: string,
  identity: StoredIdentity,
): Promise<{ keyId: string; chatKey: Uint8Array; wraps: { userId: string; wrap: ChatKeyWrap }[] }> {
  const keyId = newKeyId();
  const chatKey = generateChatKey();
  const wraps: { userId: string; wrap: ChatKeyWrap }[] = [];

  for (const userId of memberUserIds) {
    // Always the member's current public key (fetched now), never a cached one:
    // a member who regenerated their identity must get a wrap they can open.
    let publicKey: IdentityPublicKey;
    if (userId === currentUserId) {
      publicKey = toIdentityPublicKey(identity);
    } else {
      try {
        publicKey = await fetchIdentityPublicKey(userId);
      } catch (error) {
        throw new Error(`identity key not found for ${userId}`, { cause: error });
      }
    }
    const wrap = await wrapChatKey({
      chatKey,
      keyId,
      recipientPublicKey: publicKey,
    });
    wraps.push({ userId, wrap });
  }

  return { keyId, chatKey, wraps };
}

/**
 * The chat key (or the key version a message was encrypted with) cannot be opened with the identity on this
 * device, for example because the identity was regenerated after the key was wrapped.
 */
export class ChatKeyUnavailableError extends Error {
  constructor(message: string, options?: { cause?: unknown }) {
    super(message, options);
    this.name = "ChatKeyUnavailableError";
  }
}

export const PREVIOUS_KEY_PLACEHOLDER = "Message encrypted with a previous key";

// In-flight de-duplication. Opening a chat decrypts every message in parallel
// before any of them has populated the key cache, so without this each message
// fires its own key-wraps request. Entries live only while a request is pending
// and are removed in finally, so a failure never poisons later retries.
const inflightChatKeys = new Map<string, Promise<{ keyId: string; chatKey: Uint8Array }>>();
const inflightWraps = new Map<string, Promise<Awaited<ReturnType<typeof fetchChatKeyWraps>>>>();

function dedupe<T>(map: Map<string, Promise<T>>, key: string, start: () => Promise<T>): Promise<T> {
  const pending = map.get(key);
  if (pending) {
    return pending;
  }
  const promise = start().finally(() => {
    map.delete(key);
  });
  map.set(key, promise);
  return promise;
}

export async function ensureChatKey(
  userId: string,
  chatId: string,
  preferredKeyId?: string,
): Promise<{ keyId: string; chatKey: Uint8Array }> {
  if (preferredKeyId) {
    const cached = await getCachedChatKey(userId, chatId, preferredKeyId);
    if (cached) {
      return { keyId: preferredKeyId, chatKey: cached };
    }
  }

  return dedupe(inflightChatKeys, `${userId}:${chatId}:${preferredKeyId ?? ""}`, () =>
    resolveChatKey(userId, chatId, preferredKeyId),
  );
}

async function resolveChatKey(
  userId: string,
  chatId: string,
  preferredKeyId?: string,
): Promise<{ keyId: string; chatKey: Uint8Array }> {
  const identity = await loadStoredIdentity(userId);
  if (!identity) {
    throw new Error("missing identity keys");
  }

  let wraps = await dedupe(inflightWraps, `${userId}:${chatId}`, () => fetchChatKeyWraps(chatId));
  if (preferredKeyId && !wraps.some((item) => item.keyId === preferredKeyId)) {
    // A key version added after the first fetch (another member rotated the key). The shared in-flight result may
    // predate it, so look again with a separate request before giving up.
    wraps = await dedupe(inflightWraps, `${userId}:${chatId}:refresh`, () => fetchChatKeyWraps(chatId));
  }
  if (wraps.length === 0) {
    throw new Error("chat key wraps not found");
  }

  // A message names its key version, so decrypting uses exactly that version; only sending (no keyId) falls back
  // to the newest one (the list is ordered oldest first).
  const selected = preferredKeyId
    ? wraps.find((item) => item.keyId === preferredKeyId)
    : wraps[wraps.length - 1];
  if (!selected) {
    throw new ChatKeyUnavailableError(`chat key version ${preferredKeyId} is not available`);
  }

  let chatKey: Uint8Array;
  try {
    chatKey = await unwrapChatKey({ wrap: selected.wrap, identity });
  } catch (error) {
    throw new ChatKeyUnavailableError("chat key cannot be opened with the identity on this device", {
      cause: error,
    });
  }
  await cacheChatKey(userId, chatId, selected.keyId, chatKey);
  return { keyId: selected.keyId, chatKey };
}

/**
 * Adds a new chat key version wrapped for every current member's current public key and uploads it. Older
 * versions and wraps are untouched, so members whose keys did not change keep reading older messages.
 */
async function rotateChatKey(userId: string, chatId: string): Promise<{ keyId: string; chatKey: Uint8Array }> {
  const identity = await loadStoredIdentity(userId);
  if (!identity) {
    throw new Error("missing identity keys");
  }

  const members = await getChatMembers(chatId);
  const created = await createWrappedChatKey(
    members.map((member) => member.id),
    userId,
    identity,
  );
  await uploadChatKeyVersion(chatId, created.keyId, created.wraps);
  await cacheChatKey(userId, chatId, created.keyId, created.chatKey);
  return { keyId: created.keyId, chatKey: created.chatKey };
}

/**
 * Prepares adding members: one fresh chat key version wrapped for every current member and every new user. No
 * older version is opened, so new members cannot read earlier messages. Nothing is sent here; the caller posts
 * the result. Fails naming the user when any identity key is missing.
 */
export async function prepareAddMembers(
  userId: string,
  chatId: string,
  newMemberIds: string[],
): Promise<{ keyId: string; wraps: { userId: string; wrap: ChatKeyWrap }[] }> {
  const identity = await loadStoredIdentity(userId);
  if (!identity) {
    throw new Error("missing identity keys");
  }

  const members = await getChatMembers(chatId);
  const recipients = [...new Set([...members.map((member) => member.id), userId, ...newMemberIds])];
  const created = await createWrappedChatKey(recipients, userId, identity);
  await cacheChatKey(userId, chatId, created.keyId, created.chatKey);
  return { keyId: created.keyId, wraps: created.wraps };
}

export async function encryptOutgoingMessage(input: {
  plaintext: string;
  chatId: string;
  senderUserId: string;
  keyId?: string;
}): Promise<string> {
  let resolved: { keyId: string; chatKey: Uint8Array };
  try {
    resolved = await ensureChatKey(input.senderUserId, input.chatId, input.keyId);
  } catch (error) {
    if (input.keyId || !(error instanceof ChatKeyUnavailableError)) {
      throw error;
    }
    console.error("chat key unavailable on this device, adding a new key version", error);
    resolved = await rotateChatKey(input.senderUserId, input.chatId);
  }
  const { keyId, chatKey } = resolved;
  const envelope = await encryptMessage({
    plaintext: input.plaintext,
    keyId,
    chatKey,
  });
  return serializeMessageEnvelope(envelope);
}

export async function decryptIncomingMessage(message: Message, viewerUserId: string): Promise<string> {
  const envelope = parseMessageEnvelope(message.data);
  if (!envelope) {
    return message.data;
  }

  const { chatKey } = await ensureChatKey(viewerUserId, message.chatId, envelope.keyId);
  return decryptMessage({
    envelope,
    chatKey,
  });
}

export async function rememberCreatedChatKey(
  userId: string,
  chatId: string,
  keyId: string,
  chatKey: Uint8Array,
): Promise<void> {
  await cacheChatKey(userId, chatId, keyId, chatKey);
}
