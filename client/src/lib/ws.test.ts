import { afterEach, describe, expect, it } from "vitest";
import { chatWebSocketURL, buildHeartbeatMessage, isHeartbeatAckEvent, parseChatAddedEvent, parseChatUnreadEvent } from "@/lib/ws";

afterEach(() => {
  localStorage.clear();
});

describe("buildHeartbeatMessage", () => {
  it("builds a heartbeat payload", () => {
    expect(buildHeartbeatMessage()).toBe(JSON.stringify({ type: "heartbeat" }));
  });
});

describe("isHeartbeatAckEvent", () => {
  it("detects heartbeat ack payloads", () => {
    expect(isHeartbeatAckEvent(JSON.stringify({ type: "heartbeat.ack" }))).toBe(true);
    expect(isHeartbeatAckEvent(JSON.stringify({ type: "heartbeat" }))).toBe(false);
    expect(isHeartbeatAckEvent("not-json")).toBe(false);
  });
});

describe("chatWebSocketURL", () => {
  it("returns null when the user is not signed in", () => {
    expect(chatWebSocketURL()).toBeNull();
  });

  it("builds a websocket url with the stored id token", () => {
    localStorage.setItem(
      "messenger_id_token",
      "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0IiwiZXhwIjo5OTk5OTk5OTk5fQ.signature",
    );

    expect(chatWebSocketURL()).toBe(
      "ws://localhost:3000/api/v1/ws?token=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0IiwiZXhwIjo5OTk5OTk5OTk5fQ.signature",
    );
  });

  it("returns null when only an access token is stored", () => {
    localStorage.setItem("messenger_access_token", "ya29.access-token");

    expect(chatWebSocketURL()).toBeNull();
  });
});

describe("parseChatUnreadEvent", () => {
  it("parses unread chats in a websocket payload", () => {
    expect(
      parseChatUnreadEvent(
        JSON.stringify({
          type: "chat.unread",
          chats: [
            {
              chatId: "node/chat-1",
              unreadMessageCount: 3,
              updatedAt: "2026-06-11T12:00:00.000Z",
            },
            {
              chatId: "node/chat-2",
              unreadMessageCount: 1,
              updatedAt: "2026-06-10T12:00:00.000Z",
            },
          ],
        }),
      ),
    ).toEqual({
      type: "chat.unread",
      chats: [
        {
          chatId: "node/chat-1",
          unreadMessageCount: 3,
          updatedAt: "2026-06-11T12:00:00.000Z",
        },
        {
          chatId: "node/chat-2",
          unreadMessageCount: 1,
          updatedAt: "2026-06-10T12:00:00.000Z",
        },
      ],
    });
  });

  it("accepts an empty chats list", () => {
    expect(
      parseChatUnreadEvent(JSON.stringify({ type: "chat.unread", chats: [] })),
    ).toEqual({
      type: "chat.unread",
      chats: [],
    });
  });

  it("returns null for invalid payloads", () => {
    expect(parseChatUnreadEvent("not-json")).toBeNull();
    expect(parseChatUnreadEvent(JSON.stringify({ type: "other" }))).toBeNull();
    expect(
      parseChatUnreadEvent(
        JSON.stringify({
          type: "chat.unread",
          chats: [{ chatId: "node/chat-1" }],
        }),
      ),
    ).toBeNull();
  });
});

describe("parseChatAddedEvent", () => {
  it("parses chat added websocket payload", () => {
    expect(
      parseChatAddedEvent(
        JSON.stringify({
          type: "chat.added",
          chatId: "node/chat-1",
        }),
      ),
    ).toEqual({
      type: "chat.added",
      chatId: "node/chat-1",
    });
  });

  it("returns null for invalid payloads", () => {
    expect(parseChatAddedEvent("not-json")).toBeNull();
    expect(parseChatAddedEvent(JSON.stringify({ type: "chat.unread" }))).toBeNull();
    expect(parseChatAddedEvent(JSON.stringify({ type: "chat.added" }))).toBeNull();
  });
});
