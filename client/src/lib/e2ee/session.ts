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
import {
  cacheChatKey,
  fetchChatKeyWraps,
  fetchIdentityPublicKey,
  getCachedChatKey,
  loadStoredIdentity,
  saveStoredIdentity,
  uploadIdentityKey,
} from "@/lib/e2ee/storage";
import type { ChatKeyWrap, StoredIdentity } from "@/lib/e2ee/types";
import type { Message } from "@/lib/messages";

export type EnsureIdentityKeysResult = {
  identity: StoredIdentity;
  generated: boolean;
};

export async function ensureIdentityKeys(userId: string): Promise<EnsureIdentityKeysResult> {
  let identity = await loadStoredIdentity(userId);
  let generated = false;
  if (!identity) {
    identity = generateIdentityKeyPair();
    generated = true;
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
  return { identity, generated };
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

  const keyId = newKeyId();
  const chatKey = generateChatKey();
  const wraps: { userId: string; wrap: ChatKeyWrap }[] = [];

  for (const userId of input.memberUserIds) {
    const publicKey =
      userId === input.currentUserId
        ? toIdentityPublicKey(identity)
        : await fetchIdentityPublicKey(userId);
    const wrap = await wrapChatKey({
      chatKey,
      keyId,
      recipientPublicKey: publicKey,
    });
    wraps.push({ userId, wrap });
  }

  return { keyId, chatKey, wraps };
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

  const identity = await loadStoredIdentity(userId);
  if (!identity) {
    throw new Error("missing identity keys");
  }

  const wraps = await fetchChatKeyWraps(chatId);
  if (wraps.length === 0) {
    throw new Error("chat key wraps not found");
  }

  const selected = preferredKeyId
    ? wraps.find((item) => item.keyId === preferredKeyId) ?? wraps[wraps.length - 1]
    : wraps[wraps.length - 1];
  const chatKey = await unwrapChatKey({
    wrap: selected.wrap,
    identity,
  });
  await cacheChatKey(userId, chatId, selected.keyId, chatKey);
  return { keyId: selected.keyId, chatKey };
}

export async function encryptOutgoingMessage(input: {
  plaintext: string;
  chatId: string;
  senderUserId: string;
  keyId?: string;
}): Promise<string> {
  const { keyId, chatKey } = await ensureChatKey(input.senderUserId, input.chatId, input.keyId);
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
