import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChatsPage } from "@/app/ChatsPage";

vi.mock("@/hooks/useAuth", () => ({
  useAuth: vi.fn(),
}));

vi.mock("@/lib/chats", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/chats")>();
  return {
    ...actual,
    fetchChats: vi.fn(),
  };
});

vi.mock("@/lib/e2ee/session", () => ({
  decryptIncomingMessage: vi.fn(async (message: { data: string }) => message.data),
  encryptOutgoingMessage: vi.fn(async ({ plaintext }: { plaintext: string }) => plaintext),
}));

vi.mock("@/lib/messages", () => ({
  fetchMessages: vi.fn(),
  createMessage: vi.fn(),
  markMessageRead: vi.fn(),
}));

vi.mock("@/hooks/useChatSocket", () => ({
  useChatSocket: vi.fn(),
}));

import { useAuth } from "@/hooks/useAuth";
import { useChatSocket } from "@/hooks/useChatSocket";
import { fetchChats } from "@/lib/chats";
import { createMessage, fetchMessages } from "@/lib/messages";
import type { ChatUnreadEvent } from "@/lib/ws";

let onChatUnread: ((event: ChatUnreadEvent) => void) | undefined;

vi.mocked(useChatSocket).mockImplementation((_enabled, callback) => {
  onChatUnread = callback;
  return "open";
});

const authUser = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  subject: "google-subject-1",
  provider: "google",
  oidcProviderId: "11111111-1111-1111-1111-111111111111",
};

const sampleChat = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "General",
  disappearAfterMinutes: 86400,
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

const sampleMessage = {
  id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  chatId: sampleChat.id,
  userId: authUser.id,
  data: "hello",
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

describe("ChatsPage", () => {
  beforeEach(() => {
    onChatUnread = undefined;
    vi.mocked(fetchMessages).mockReset();
    vi.mocked(fetchChats).mockReset();
  });

  it("renders chats and messages in a split layout", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([sampleChat]);
    vi.mocked(fetchMessages).mockResolvedValue([sampleMessage]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(screen.getByRole("option", { name: /General/i })).toHaveAttribute(
        "aria-selected",
        "true",
      );
    });

    await waitFor(() => {
      expect(screen.getByText("hello")).toBeInTheDocument();
    });
    expect(fetchMessages).toHaveBeenCalledWith(sampleChat.id);
    expect(screen.getByLabelText("Message")).toBeInTheDocument();
  });

  it("sends a new message", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([sampleChat]);
    vi.mocked(fetchMessages).mockResolvedValue([]);
    vi.mocked(createMessage).mockResolvedValue(sampleMessage);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Message")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Message"), {
      target: { value: "hello" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => {
      expect(createMessage).toHaveBeenCalledWith({
        chatId: sampleChat.id,
        userId: authUser.id,
        data: "hello",
      });
    });
  });

  it("shows a New chat button when there are no chats", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(screen.getByText("No chats yet.")).toBeInTheDocument();
    });

    expect(screen.getByRole("button", { name: "New chat" })).toBeInTheDocument();
    expect(screen.getByText("Select a chat to view messages.")).toBeInTheDocument();
  });

  it("prompts guests to sign in", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });

    render(<ChatsPage />);

    expect(screen.getByText("Sign in to view your chats.")).toBeInTheDocument();
  });

  it("reloads messages when the open chat unread count changes", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([
      { ...sampleChat, unreadMessageCount: 0 },
    ]);
    vi.mocked(fetchMessages)
      .mockResolvedValueOnce([sampleMessage])
      .mockResolvedValueOnce([
        sampleMessage,
        {
          ...sampleMessage,
          id: "cccccccc-cccc-cccc-cccc-cccccccccccc",
          data: "new message",
        },
      ]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(screen.getByText("hello")).toBeInTheDocument();
    });
    expect(fetchMessages).toHaveBeenCalledTimes(1);

    await act(async () => {
      onChatUnread?.({
        type: "chat.unread",
        chats: [
          {
            chatId: sampleChat.id,
            unreadMessageCount: 1,
            updatedAt: "2026-06-11T12:05:00.000Z",
          },
        ],
      });
    });

    await waitFor(() => {
      expect(fetchMessages).toHaveBeenCalledTimes(2);
    });
    await waitFor(() => {
      expect(screen.getByText("new message")).toBeInTheDocument();
    });
  });
});
