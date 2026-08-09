import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";
import { DEFAULT_DISAPPEAR_AFTER_MINUTES } from "@/lib/disappear";
import type { ChatKeyWrap } from "@/lib/e2ee/types";
import { scopeResourceIds } from "@/lib/scopedId";

export type Chat = {
  id: string;
  name?: string;
  disappearAfterMinutes: number;
  createdAt: string;
  updatedAt: string;
  unreadMessageCount?: number;
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
