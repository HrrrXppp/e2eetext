import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  NOTIFICATION_CHAT_SELECT_EVENT,
  getNotificationPermission,
  isNotificationSupported,
  notifyNewMessage,
  requestNotificationPermission,
} from "@/lib/notifications";

const chat = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "General",
  disappearAfterMinutes: 86400,
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

type FakeNotification = { onclick: (() => void) | null; close: ReturnType<typeof vi.fn> };

let instances: FakeNotification[];
let constructed: Array<[string, NotificationOptions | undefined]>;
let permission: NotificationPermission;
let requestResult: NotificationPermission;

class FakeNotificationCtor {
  static get permission() {
    return permission;
  }
  static requestPermission = vi.fn(async () => requestResult);
  onclick: (() => void) | null = null;
  close = vi.fn();
  constructor(title: string, options?: NotificationOptions) {
    constructed.push([title, options]);
    instances.push(this);
  }
}

describe("notifications", () => {
  beforeEach(() => {
    instances = [];
    constructed = [];
    permission = "default";
    requestResult = "granted";
    vi.stubGlobal("Notification", FakeNotificationCtor);
    vi.stubGlobal("isSecureContext", true);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("is unsupported without the API, outside secure contexts, or on mobile", () => {
    expect(isNotificationSupported()).toBe(true);

    vi.stubGlobal("isSecureContext", false);
    expect(isNotificationSupported()).toBe(false);
    vi.stubGlobal("isSecureContext", true);

    vi.spyOn(window.navigator, "userAgent", "get").mockReturnValue("Mozilla/5.0 (Linux; Android 14)");
    expect(isNotificationSupported()).toBe(false);
  });

  it("reports and requests permission", async () => {
    expect(getNotificationPermission()).toBe("default");
    await expect(requestNotificationPermission()).resolves.toBe("granted");
  });

  it("does not notify unless permission is granted", () => {
    notifyNewMessage(chat);
    expect(constructed).toHaveLength(0);
  });

  it("notifies with a generic body and per-chat tag", () => {
    permission = "granted";
    notifyNewMessage(chat);
    expect(constructed).toEqual([
      ["New message in General", { tag: `chat-${chat.id}` }],
    ]);
  });

  it("falls back to Unnamed chat", () => {
    permission = "granted";
    notifyNewMessage({ ...chat, name: undefined });
    expect(constructed[0][0]).toBe("New message in Unnamed chat");
  });

  it("dispatches the select event on click", () => {
    permission = "granted";
    const listener = vi.fn();
    window.addEventListener(NOTIFICATION_CHAT_SELECT_EVENT, listener);
    vi.spyOn(window, "focus").mockImplementation(() => {});
    notifyNewMessage(chat);
    instances[0].onclick?.();
    window.removeEventListener(NOTIFICATION_CHAT_SELECT_EVENT, listener);

    expect(listener).toHaveBeenCalledTimes(1);
    expect((listener.mock.calls[0][0] as CustomEvent).detail).toEqual({ chatId: chat.id });
    expect(instances[0].close).toHaveBeenCalled();
  });

  it("swallows constructor errors", () => {
    permission = "granted";
    vi.stubGlobal(
      "Notification",
      Object.assign(
        function Throwing() {
          throw new TypeError("Illegal constructor");
        },
        { permission: "granted" },
      ),
    );
    expect(() => notifyNewMessage(chat)).not.toThrow();
  });
});
