import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";
import { DEFAULT_DISAPPEAR_AFTER_MINUTES } from "@/lib/disappear";
import type { ChatKeyWrap } from "@/lib/e2ee/types";
import { scopeResourceIds } from "@/lib/scopedId";
import type { User } from "@/lib/users";

export type Chat = {
  id: string;
  name?: string;
  disappearAfterMinutes: number;
  createdAt: string;
  updatedAt: string;
  unreadMessageCount?: number;
  isAdmin?: boolean;
};

export type ChatMember = User & {
  isAdmin: boolean;
};

export function formatChatUpdatedAt(value: string): string {
  return new Date(value).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export async function fetchChats(userId: string): Promise<Chat[]> {
  const params = new URLSearchParams({ user_id: userId });

  const response = await fetch(`${API_V1}/chat?${params.toString()}`, {
    headers: authHeaders(),
  });

  if (!response.ok) {
    throw new Error("failed to fetch chats");
  }

  return response.json() as Promise<Chat[]>;
}

type CreateChatInput = {
  name: string;
  usersUids: string[];
  keyId: string;
  wraps: { userId: string; wrap: ChatKeyWrap }[];
  disappearAfterMinutes?: number;
};

export async function createChat(input: CreateChatInput): Promise<Chat> {
  const scoped = scopeResourceIds([
    ...input.usersUids,
    ...input.wraps.map((item) => item.userId),
  ]);
  const usersUids = scoped.slice(0, input.usersUids.length);
  const wraps = input.wraps.map((item, index) => ({
    user_id: scoped[input.usersUids.length + index],
    wrap: item.wrap,
  }));

  const response = await fetch(`${API_V1}/chat`, {
    method: "POST",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: input.name,
      users_uids: usersUids,
      disappear_after_minutes: input.disappearAfterMinutes ?? DEFAULT_DISAPPEAR_AFTER_MINUTES,
      e2ee: {
        key_id: input.keyId,
        wraps,
      },
    }),
  });

  if (!response.ok) {
    throw new Error("failed to create chat");
  }

  return response.json() as Promise<Chat>;
}

export async function getChatMembers(chatId: string): Promise<ChatMember[]> {
  const response = await fetch(`${API_V1}/chat/${chatId}/members`, {
    headers: authHeaders(),
  });

  if (response.status === 403) {
    throw new Error("chat access denied");
  }

  if (!response.ok) {
    throw new Error("failed to fetch chat members");
  }

  return response.json() as Promise<ChatMember[]>;
}

export async function removeMember(chatId: string, userId: string): Promise<void> {
  const response = await fetch(`${API_V1}/chat/${chatId}/members/${userId}`, {
    method: "DELETE",
    headers: authHeaders(),
  });

  if (response.status === 403) {
    throw new Error("chat admin required");
  }

  if (!response.ok) {
    throw new Error("failed to remove chat member");
  }
}

async function serverErrorMessage(response: Response): Promise<string> {
  try {
    const data = (await response.json()) as { error?: unknown };
    return typeof data.error === "string" ? data.error : "";
  } catch {
    return "";
  }
}

/**
 * Adds users to a chat with one freshly generated key version. The wraps must cover every current member and
 * every new user, because the server only accepts a complete set.
 */
export async function addMembers(
  chatId: string,
  userIds: string[],
  keyId: string,
  wraps: { userId: string; wrap: ChatKeyWrap }[],
): Promise<void> {
  const scoped = scopeResourceIds([...userIds, ...wraps.map((item) => item.userId)]);

  const response = await fetch(`${API_V1}/chat/${chatId}/members`, {
    method: "POST",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      user_ids: scoped.slice(0, userIds.length),
      key_id: keyId,
      wraps: wraps.map((item, index) => ({
        user_id: scoped[userIds.length + index],
        wrap: item.wrap,
      })),
    }),
  });

  if (response.status === 403) {
    throw new Error("chat admin required");
  }

  if (response.status === 409) {
    const detail = await serverErrorMessage(response);
    throw new Error(detail ? `already a member: ${detail}` : "already a member");
  }

  if (!response.ok) {
    throw new Error("failed to add chat members");
  }
}
