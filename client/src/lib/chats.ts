import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";
import { scopeResourceIds } from "@/lib/scopedId";

export type Chat = {
  id: string;
  name?: string;
  adminUserId: string;
  kemPublicKey: string;
  wrappedChatPrivateKey: string;
  kemCiphertext: string;
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

export type ChatMemberInput = {
  userId: string;
  wrappedChatPrivateKey: string;
  kemCiphertext: string;
};

type CreateChatInput = {
  name: string;
  kemPublicKey: string;
  members: ChatMemberInput[];
};

export async function createChat(input: CreateChatInput): Promise<Chat> {
  const scopedUserIds = scopeResourceIds(input.members.map((member) => member.userId));

  const response = await fetch(`${API_V1}/chat`, {
    method: "POST",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      name: input.name,
      kem_public_key: input.kemPublicKey,
      members: input.members.map((member, index) => ({
        user_id: scopedUserIds[index],
        wrapped_chat_private_key: member.wrappedChatPrivateKey,
        kem_ciphertext: member.kemCiphertext,
      })),
    }),
  });

  if (!response.ok) {
    throw new Error("failed to create chat");
  }

  return response.json() as Promise<Chat>;
}
