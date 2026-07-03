import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";
import { scopeResourceIds } from "@/lib/scopedId";

export type Chat = {
  id: string;
  name?: string;
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
};

export async function createChat(input: CreateChatInput): Promise<Chat> {
  const usersUids = scopeResourceIds(input.usersUids);

  const response = await fetch(`${API_V1}/chat`, {
    method: "POST",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: input.name,
      users_uids: usersUids,
    }),
  });

  if (!response.ok) {
    throw new Error("failed to create chat");
  }

  return response.json() as Promise<Chat>;
}
