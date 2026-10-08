import type { Chat } from "@/lib/chats";

// Fired on `window` when a desktop notification is clicked while this tab
// is already open, so ChatsPage can select the chat without a reload.
export const NOTIFICATION_CHAT_SELECT_EVENT = "messenger:notification-chat-select";

export type NotificationChatSelectDetail = {
  chatId: string;
};

// Fallback for the (rare) case where the notification is clicked after the
// tab was fully closed and the browser has to open/focus a fresh one: the
// click handler can't dispatch a DOM event to a document that doesn't exist
// yet, so it writes here instead and ChatsPage consumes it on mount.
const PENDING_CHAT_ID_KEY = "messenger_pending_notification_chat_id";

function isMobileUserAgent(): boolean {
  const ua = window.navigator?.userAgent ?? "";
  // Android Chrome throws on the plain `new Notification(...)` constructor
  // (it requires ServiceWorkerRegistration.showNotification instead), and
  // iOS Safari only supports web notifications for a site installed to the
  // Home Screen as a PWA -- a normal tab gets none. Both are out of scope
  // for this (desktop-only) pass, so this feature-detects them out rather
  // than risking a thrown exception or a browser that silently never shows
  // anything. See the plan on issue #48 for the full reasoning.
  return /Android|iPhone|iPad|iPod/i.test(ua);
}

export function isNotificationSupported(): boolean {
  if (typeof window === "undefined" || !("Notification" in window)) {
    return false;
  }

  if (!window.isSecureContext) {
    return false;
  }

  return !isMobileUserAgent();
}

export function getNotificationPermission(): NotificationPermission {
  if (!isNotificationSupported()) {
    return "denied";
  }

  return Notification.permission;
}

export async function requestNotificationPermission(): Promise<NotificationPermission> {
  if (!isNotificationSupported()) {
    return "denied";
  }

  return Notification.requestPermission();
}

function setPendingNotificationChatId(chatId: string): void {
  try {
    sessionStorage.setItem(PENDING_CHAT_ID_KEY, chatId);
  } catch {
    // Private browsing / storage quota -- the same-tab custom-event path
    // (the common case) still works without this fallback.
  }
}

export function consumePendingNotificationChatId(): string | null {
  try {
    const chatId = sessionStorage.getItem(PENDING_CHAT_ID_KEY);
    if (chatId) {
      sessionStorage.removeItem(PENDING_CHAT_ID_KEY);
    }
    return chatId;
  } catch {
    return null;
  }
}

/**
 * Fires a desktop OS notification for a new message in `chat`. No-ops
 * unless permission has already been granted. Deliberately generic copy
 * only (chat name, no message preview/sender) -- message content is E2EE
 * plaintext that shouldn't land in an OS notification center or log.
 */
export function notifyNewMessage(chat: Pick<Chat, "id" | "name">): void {
  if (getNotificationPermission() !== "granted") {
    return;
  }

  const label = chat.name?.trim() || "Unnamed chat";

  let notification: Notification;
  try {
    notification = new Notification(`New message in ${label}`, {
      tag: `chat-${chat.id}`,
    });
  } catch {
    // Defensive fallback: isNotificationSupported() should have filtered
    // out browsers where the constructor throws, but fail safe rather than
    // crash the caller (handleChatUnread) if one slips through.
    return;
  }

  notification.onclick = () => {
    window.focus();
    setPendingNotificationChatId(chat.id);
    window.dispatchEvent(
      new CustomEvent<NotificationChatSelectDetail>(NOTIFICATION_CHAT_SELECT_EVENT, {
        detail: { chatId: chat.id },
      }),
    );
    notification.close();
  };
}
