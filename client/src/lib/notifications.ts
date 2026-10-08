import type { Chat } from "@/lib/chats";

export const NOTIFICATION_CHAT_SELECT_EVENT = "messenger:notification-chat-select";

export type NotificationPermissionState = "default" | "granted" | "denied";

const MOBILE_USER_AGENT = /Android|iPhone|iPad|iPod/i;

// Desktop only: mobile browsers need a service worker + Web Push (separate issue).
export function isNotificationSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "Notification" in window &&
    window.isSecureContext &&
    !MOBILE_USER_AGENT.test(window.navigator.userAgent)
  );
}

export function getNotificationPermission(): NotificationPermissionState {
  if (!isNotificationSupported()) {
    return "denied";
  }
  return window.Notification.permission;
}

export async function requestNotificationPermission(): Promise<NotificationPermissionState> {
  if (!isNotificationSupported()) {
    return "denied";
  }
  try {
    return await window.Notification.requestPermission();
  } catch {
    return getNotificationPermission();
  }
}

export function chatNotificationLabel(chat: Chat): string {
  return chat.name?.trim() || "Unnamed chat";
}

// Body is intentionally generic: message plaintext must not reach OS notification centers.
export function notifyNewMessage(chat: Chat): void {
  if (getNotificationPermission() !== "granted") {
    return;
  }

  try {
    const notification = new window.Notification(`New message in ${chatNotificationLabel(chat)}`, {
      tag: `chat-${chat.id}`,
    });
    notification.onclick = () => {
      window.focus();
      window.dispatchEvent(
        new CustomEvent(NOTIFICATION_CHAT_SELECT_EVENT, { detail: { chatId: chat.id } }),
      );
      notification.close();
    };
  } catch {
    // Some browsers throw on the constructor; notifications are best-effort.
  }
}
