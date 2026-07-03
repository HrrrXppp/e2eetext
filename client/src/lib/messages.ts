import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";

export type Message = {
  id: string;
  chatId: string;
  userId: string;
  userName?: string;
  data: string;
  createdAt: string;
  updatedAt: string;
  unread?: boolean;
};

export async function fetchMessages(chatId: string): Promise<Message[]> {
  const params = new URLSearchParams({ chat_id: chatId });

  const response = await fetch(`${API_V1}/message?${params.toString()}`, {
    headers: authHeaders(),
  });

  if (!response.ok) {
    throw new Error("failed to fetch messages");
  }

  return response.json() as Promise<Message[]>;
}

type CreateMessageInput = {
  chatId: string;
  userId: string;
  data: string;
};

export async function createMessage(input: CreateMessageInput): Promise<Message> {
  const response = await fetch(`${API_V1}/message`, {
    method: "POST",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      chat_id: input.chatId,
      user_id: input.userId,
      data: input.data,
    }),
  });

  if (!response.ok) {
    throw new Error("failed to create message");
  }

  return response.json() as Promise<Message>;
}

export async function markMessageRead(messageId: string): Promise<Message> {
  const response = await fetch(`${API_V1}/message/${messageId}`, {
    method: "PATCH",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ unread: false }),
  });

  if (!response.ok) {
    throw new Error("failed to mark message read");
  }

  return response.json() as Promise<Message>;
}
