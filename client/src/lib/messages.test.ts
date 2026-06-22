import { afterEach, describe, expect, it, vi } from "vitest";
import { createMessage, fetchMessages, markMessageRead } from "@/lib/messages";

const testNodeId = "99999999-9999-9999-9999-999999999999";
const scopedChatId = `${testNodeId}/11111111-1111-1111-1111-111111111111`;
const scopedUserId = `${testNodeId}/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb`;

const sampleMessage = {
  id: `${testNodeId}/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`,
  chatId: scopedChatId,
  userId: scopedUserId,
  data: "hello",
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("fetchMessages", () => {
  it("requests messages for a chat", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [sampleMessage],
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(fetchMessages(scopedChatId)).resolves.toEqual([
      sampleMessage,
    ]);

    expect(fetchMock).toHaveBeenCalledWith(
      `/api/v1/message?chat_id=${encodeURIComponent(scopedChatId)}`,
      { headers: { Authorization: "Bearer token" } },
    );
  });
});

describe("createMessage", () => {
  it("creates a message", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => sampleMessage,
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      createMessage({
        chatId: scopedChatId,
        userId: scopedUserId,
        data: "hello",
      }),
    ).resolves.toEqual(sampleMessage);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/message", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        chat_id: scopedChatId,
        user_id: scopedUserId,
        data: "hello",
      }),
    });
  });
});

describe("markMessageRead", () => {
  it("marks a message as read", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ ...sampleMessage, unread: false }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(markMessageRead(sampleMessage.id)).resolves.toEqual({
      ...sampleMessage,
      unread: false,
    });

    expect(fetchMock).toHaveBeenCalledWith(`/api/v1/message/${sampleMessage.id}`, {
      method: "PATCH",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ unread: false }),
    });
  });
});
