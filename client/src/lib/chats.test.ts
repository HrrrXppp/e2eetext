import { afterEach, describe, expect, it, vi } from "vitest";
import { createChat, fetchChats, formatChatUpdatedAt } from "@/lib/chats";

const testNodeId = "99999999-9999-9999-9999-999999999999";
const scopedUserId = `${testNodeId}/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa`;

const sampleChat = {
  id: `${testNodeId}/11111111-1111-1111-1111-111111111111`,
  name: "General",
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
  const userC = "cccccccc-cccc-cccc-cccc-cccccccccccc";
  const userD = "dddddddd-dddd-dddd-dddd-dddddddddddd";
  const scopedUserB = `${testNodeId}/${userB}`;
  const scopedUserC = `${testNodeId}/${userC}`;
  const scopedUserD = `${testNodeId}/${userD}`;

  function mockCreateChat(response: Chat = sampleChat) {
    localStorage.setItem("messenger_id_token", "token");
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => response,
    });
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
  }

  function expectCreateChatBody(fetchMock: ReturnType<typeof vi.fn>, usersUids: string[], name = "General") {
    expect(fetchMock).toHaveBeenCalledWith("/api/v1/chat", {
      method: "POST",
      headers: {
        Authorization: "Bearer token",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name, users_uids: usersUids }),
    });
  }

  it("creates a solo chat with only the current user", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "Notes",
        usersUids: [userA],
      }),
    ).resolves.toEqual(sampleChat);

    expectCreateChatBody(fetchMock, [userA], "Notes");
  });

  it("creates a chat with the current user and one member", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "General",
        usersUids: [userA, userB],
      }),
    ).resolves.toEqual(sampleChat);

    expectCreateChatBody(fetchMock, [userA, scopedUserB]);
  });

  it("creates a chat with the current user and two members", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "Project",
        usersUids: [userA, userB, userC],
      }),
    ).resolves.toEqual(sampleChat);

    expectCreateChatBody(fetchMock, [userA, scopedUserB, scopedUserC], "Project");
  });

  it("creates a chat with the current user and three members", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "Team",
        usersUids: [userA, userB, userC, userD],
      }),
    ).resolves.toEqual(sampleChat);

    expectCreateChatBody(fetchMock, [userA, scopedUserB, scopedUserC, scopedUserD], "Team");
  });

  it("keeps already scoped member ids unchanged", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "General",
        usersUids: [userA, scopedUserB, scopedUserC],
      }),
    ).resolves.toEqual(sampleChat);

    expectCreateChatBody(fetchMock, [userA, scopedUserB, scopedUserC]);
  });

  it("scopes members using node id from any scoped user in the list", async () => {
    const fetchMock = mockCreateChat();

    await expect(
      createChat({
        name: "General",
        usersUids: [userB, userA, userC],
      }),
    ).resolves.toEqual(sampleChat);

    expectCreateChatBody(fetchMock, [scopedUserB, userA, scopedUserC]);
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
        usersUids: [userA, userB],
      }),
    ).rejects.toThrow("failed to create chat");
  });
});
