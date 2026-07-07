import { afterEach, describe, expect, it, vi } from "vitest";
import { createChat, fetchChats, formatChatUpdatedAt, type Chat } from "@/lib/chats";

const testNodeId = "99999999-9999-9999-9999-999999999999";
const scopedUserId = `${testNodeId}/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`;

const sampleChat: Chat = {
  id: `${testNodeId}/11111111-1111-1111-1111-111111111111`,
  name: "General",
  adminUserId: scopedUserId,
  kemPublicKey: "chat-kem-public-key",
  wrappedChatPrivateKey: "wrapped-chat-private-key",
  kemCiphertext: "chat-kem-ciphertext",
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("formatChatUpdatedAt", () => {
  it("formats updated time for chat info", () => {
    const formatted = formatChatUpdatedAt("2026-06-11T15:30:00.000Z");
    expect(formatted).toContain("Jun");
    expect(formatted).toContain("11");
  });
});

describe("fetchChats", () => {
  it("requests chats for the current user", async () => {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [sampleChat],
    });
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      fetchChats(`${testNodeId}/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`),
    ).resolves.toEqual([sampleChat]);

    expect(fetchMock).toHaveBeenCalledWith(
      `/api/v1/chat?user_id=${encodeURIComponent(`${testNodeId}/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`)}`,
      { headers: { Authorization: "Bearer token" } },
    );
  });
});

describe("createChat", () => {
  const userA = scopedUserId;
  const userB = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb";
  const scopedUserB = `${testNodeId}/${userB}`;

  function mockCreateChat(response: Chat = sampleChat) {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => response,
    });
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
  }

  function member(userId: string, suffix: string) {
    return { userId, wrappedChatPrivateKey: `wrapped-${suffix}`, kemCiphertext: `ct-${suffix}` };
  }

  it("creates a solo chat with only the current user", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "Notes",
        kemPublicKey: "chat-kem-public-key",
        members: [member(userA, "a")],
      }),
    ).resolves.toEqual(sampleChat);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chat", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        name: "Notes",
        kem_public_key: "chat-kem-public-key",
        members: [{ user_id: userA, wrapped_chat_private_key: "wrapped-a", kem_ciphertext: "ct-a" }],
      }),
    });
  });

  it("creates a chat with the current user and one member, scoping unscoped ids", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "General",
        kemPublicKey: "chat-kem-public-key",
        members: [member(userA, "a"), member(userB, "b")],
      }),
    ).resolves.toEqual(sampleChat);

    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chat", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        name: "General",
        kem_public_key: "chat-kem-public-key",
        members: [
          { user_id: userA, wrapped_chat_private_key: "wrapped-a", kem_ciphertext: "ct-a" },
          { user_id: scopedUserB, wrapped_chat_private_key: "wrapped-b", kem_ciphertext: "ct-b" },
        ],
      }),
    });
  });

  it("keeps already scoped member ids unchanged", async () => {
    const fetchMock = mockCreateChat();

    await createChat({
      name: "General",
      kemPublicKey: "chat-kem-public-key",
      members: [member(userA, "a"), member(scopedUserB, "b")],
    });

    const [, init] = fetchMock.mock.calls[0];
    const body = JSON.parse(init.body as string);
    expect(body.members.map((m: { user_id: string }) => m.user_id)).toEqual([userA, scopedUserB]);
  });

  it("throws when the server rejects chat creation", async () => {
    localStorage.setItem("messenger_id_token", "token");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
      }),
    );

    await expect(
      createChat({
        name: "General",
        kemPublicKey: "chat-kem-public-key",
        members: [member(userA, "a"), member(userB, "b")],
      }),
    ).rejects.toThrow("failed to create chat");
  });
});
