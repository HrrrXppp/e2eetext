import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  consumePendingNotificationChatId,
  getNotificationPermission,
  isNotificationSupported,
  NOTIFICATION_CHAT_SELECT_EVENT,
  notifyNewMessage,
  requestNotificationPermission,
} from "@/lib/notifications";

class MockNotification {
  static permission: NotificationPermission = "default";
  static requestPermission = vi.fn(async () => MockNotification.permission);
  static instances: MockNotification[] = [];

  onclick: ((this: Notification, ev: Event) => unknown) | null = null;
  close = vi.fn();
  title: string;
  options?: NotificationOptions;

  constructor(title: string, options?: NotificationOptions) {
    this.title = title;
    this.options = options;
    MockNotification.instances.push(this);
  }
}

function lastNotification(): MockNotification {
  const instance = MockNotification.instances.at(-1);
  if (!instance) {
    throw new Error("expected a Notification to have been constructed");
  }
  return instance;
}

const DESKTOP_UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/119";
const ANDROID_UA = "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 Chrome/119 Mobile";
const IOS_UA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15";

function setUserAgent(ua: string) {
  Object.defineProperty(window.navigator, "userAgent", {
    value: ua,
    configurable: true,
  });
}

function setSecureContext(secure: boolean) {
  Object.defineProperty(window, "isSecureContext", {
    value: secure,
    configurable: true,
  });
}

describe("notifications", () => {
  const originalUserAgent = window.navigator.userAgent;

  beforeEach(() => {
    MockNotification.permission = "default";
    MockNotification.requestPermission.mockClear();
    MockNotification.instances = [];
    vi.stubGlobal("Notification", MockNotification);
    setSecureContext(true);
    setUserAgent(DESKTOP_UA);
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    setUserAgent(originalUserAgent);
  });

  describe("isNotificationSupported", () => {
    it("is true on a secure desktop browser", () => {
      expect(isNotificationSupported()).toBe(true);
    });

    it("is false without a secure context", () => {
      setSecureContext(false);
      expect(isNotificationSupported()).toBe(false);
    });

    it("is false on Android", () => {
      setUserAgent(ANDROID_UA);
      expect(isNotificationSupported()).toBe(false);
    });

    it("is false on iOS", () => {
      setUserAgent(IOS_UA);
      expect(isNotificationSupported()).toBe(false);
    });

    it("is false when the Notification API is absent", () => {
      vi.unstubAllGlobals();
      expect(isNotificationSupported()).toBe(false);
    });
  });

  describe("getNotificationPermission / requestNotificationPermission", () => {
    it("reflects Notification.permission when supported", () => {
      MockNotification.permission = "granted";
      expect(getNotificationPermission()).toBe("granted");
    });

    it("reports denied when unsupported, without touching the API", () => {
      setUserAgent(ANDROID_UA);
      expect(getNotificationPermission()).toBe("denied");
    });

    it("delegates to Notification.requestPermission when supported", async () => {
      MockNotification.requestPermission.mockResolvedValueOnce("granted");
      const result = await requestNotificationPermission();
      expect(result).toBe("granted");
      expect(MockNotification.requestPermission).toHaveBeenCalledTimes(1);
    });

    it("no-ops to denied when unsupported", async () => {
      setUserAgent(ANDROID_UA);
      const result = await requestNotificationPermission();
      expect(result).toBe("denied");
      expect(MockNotification.requestPermission).not.toHaveBeenCalled();
    });
  });

  describe("notifyNewMessage", () => {
    const chat = { id: "chat-1", name: "General" };

    it("does nothing when permission is not granted", () => {
      MockNotification.permission = "default";
      notifyNewMessage(chat);
      expect(MockNotification.instances).toHaveLength(0);
    });

    it("includes the chat name in the title and tags by chat id", () => {
      MockNotification.permission = "granted";

      notifyNewMessage(chat);

      const created = lastNotification();
      expect(created.title).toBe("New message in General");
      expect(created.options?.tag).toBe("chat-chat-1");
    });

    it("falls back to 'Unnamed chat' when the chat has no name", () => {
      MockNotification.permission = "granted";

      notifyNewMessage({ id: "chat-2", name: undefined });

      expect(lastNotification().title).toBe("New message in Unnamed chat");
    });

    it("on click: focuses the window, stashes the chat id, and dispatches the select event", () => {
      MockNotification.permission = "granted";

      const focusSpy = vi.spyOn(window, "focus").mockImplementation(() => undefined);
      const listener = vi.fn();
      window.addEventListener(NOTIFICATION_CHAT_SELECT_EVENT, listener);

      notifyNewMessage(chat);
      const created = lastNotification();
      created.onclick?.call(created as unknown as Notification, new Event("click"));

      expect(focusSpy).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledTimes(1);
      const event = listener.mock.calls[0][0] as CustomEvent<{ chatId: string }>;
      expect(event.detail).toEqual({ chatId: "chat-1" });
      expect(created.close).toHaveBeenCalledTimes(1);
      expect(consumePendingNotificationChatId()).toBe("chat-1");

      window.removeEventListener(NOTIFICATION_CHAT_SELECT_EVENT, listener);
      focusSpy.mockRestore();
    });
  });

  describe("consumePendingNotificationChatId", () => {
    it("returns null and stays empty when nothing is pending", () => {
      expect(consumePendingNotificationChatId()).toBeNull();
      expect(consumePendingNotificationChatId()).toBeNull();
    });
  });
});
