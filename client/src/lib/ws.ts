import { API_V1 } from "@/lib/api";
import { getWebSocketIdToken } from "@/lib/auth";

export const CHAT_UNREAD_EVENT = "chat.unread";
export const CHAT_ADDED_EVENT = "chat.added";
export const HEARTBEAT_EVENT = "heartbeat";
export const HEARTBEAT_ACK_EVENT = "heartbeat.ack";

export const HEARTBEAT_INTERVAL_MS = 30_000;

export type ChatUnreadItem = {
  chatId: string;
  unreadMessageCount: number;
  updatedAt: string;
};

export type ChatUnreadEvent = {
  type: typeof CHAT_UNREAD_EVENT;
  chats: ChatUnreadItem[];
};

export type ChatAddedEvent = {
  type: typeof CHAT_ADDED_EVENT;
  chatId: string;
};

export function buildChatWebSocketURL(token: string): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const params = new URLSearchParams({ token });
  return `${protocol}//${window.location.host}${API_V1}/ws?${params.toString()}`;
}

export function buildHeartbeatMessage(): string {
  return JSON.stringify({ type: HEARTBEAT_EVENT });
}

export function isHeartbeatAckEvent(data: unknown): boolean {
  if (typeof data !== "string") {
    return false;
  }

  let payload: unknown;
  try {
    payload = JSON.parse(data);
  } catch {
    return false;
  }

  if (!payload || typeof payload !== "object") {
    return false;
  }

  return (payload as { type?: string }).type === HEARTBEAT_ACK_EVENT;
}

export function chatWebSocketURL(): string | null {
  const token = getWebSocketIdToken();
  if (!token) {
    return null;
  }

  return buildChatWebSocketURL(token);
}

function parseChatUnreadItem(value: unknown): ChatUnreadItem | null {
  if (!value || typeof value !== "object") {
    return null;
  }

  const item = value as Partial<ChatUnreadItem>;
  if (
    typeof item.chatId !== "string" ||
    typeof item.unreadMessageCount !== "number" ||
    typeof item.updatedAt !== "string"
  ) {
    return null;
  }

  return {
    chatId: item.chatId,
    unreadMessageCount: item.unreadMessageCount,
    updatedAt: item.updatedAt,
  };
}

export function parseChatUnreadEvent(data: unknown): ChatUnreadEvent | null {
  if (typeof data !== "string") {
    return null;
  }

  let payload: unknown;
  try {
    payload = JSON.parse(data);
  } catch {
    return null;
  }

  if (!payload || typeof payload !== "object") {
    return null;
  }

  const event = payload as Partial<ChatUnreadEvent>;
  if (event.type !== CHAT_UNREAD_EVENT || !Array.isArray(event.chats)) {
    return null;
  }

  const chats: ChatUnreadItem[] = [];
  for (const item of event.chats) {
    const parsed = parseChatUnreadItem(item);
    if (!parsed) {
      return null;
    }
    chats.push(parsed);
  }

  return {
    type: CHAT_UNREAD_EVENT,
    chats,
  };
}

export function parseChatAddedEvent(data: unknown): ChatAddedEvent | null {
  if (typeof data !== "string") {
    return null;
  }

  let payload: unknown;
  try {
    payload = JSON.parse(data);
  } catch {
    return null;
  }

  if (!payload || typeof payload !== "object") {
    return null;
  }

  const event = payload as Partial<ChatAddedEvent>;
  if (event.type !== CHAT_ADDED_EVENT || typeof event.chatId !== "string") {
    return null;
  }

  return {
    type: CHAT_ADDED_EVENT,
    chatId: event.chatId,
  };
}
