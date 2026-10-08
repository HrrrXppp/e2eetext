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

vi.mock("@/lib/notifications", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/notifications")>();
  return {
    ...actual,
    notifyNewMessage: vi.fn(),
    consumePendingNotificationChatId: vi.fn(() => null),
  };
});

import { useAuth } from "@/hooks/useAuth";
import { useChatSocket } from "@/hooks/useChatSocket";
import { fetchChats } from "@/lib/chats";
import { createMessage, fetchMessages } from "@/lib/messages";
import {
  consumePendingNotificationChatId,
  notifyNewMessage,
  NOTIFICATION_CHAT_SELECT_EVENT,
} from "@/lib/notifications";
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

const otherChat = {
  id: "22222222-2222-2222-2222-222222222222",
  name: "Random",
  disappearAfterMinutes: 86400,
  createdAt: "2026-06-10T12:00:00.000Z",
  updatedAt: "2026-06-10T12:00:00.000Z",
};

describe("ChatsPage", () => {
  beforeEach(() => {
    onChatUnread = undefined;
    vi.mocked(fetchMessages).mockReset();
    vi.mocked(fetchChats).mockReset();
    vi.mocked(notifyNewMessage).mockReset();
    vi.mocked(consumePendingNotificationChatId).mockReset().mockReturnValue(null);
  });

  it("renders chats and messages in a split layout", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });

    render(<ChatsPage />);

    expect(screen.getByText("Sign in to view your chats.")).toBeInTheDocument();
    expect(
      screen.getByText(/E2EE Text will be sustained through/),
    ).toBeInTheDocument();
  });

  it("reloads messages when the open chat unread count changes", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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

  it("keeps the chat list and message panel as siblings after a chat is selected", async () => {
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
      expect(screen.getByText("hello")).toBeInTheDocument();
    });

    expect(screen.getByRole("listbox", { name: "Chats" })).toBeInTheDocument();
    expect(document.querySelector(".chats-page__layout")).not.toHaveClass(
      "chats-page__layout--chat-open",
    );
    expect(
      screen.queryByRole("button", { name: "Back to chats" }),
    ).not.toBeInTheDocument();
  });

  it("notifies for a background chat whose unread count increases, but not for the open one", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([
      { ...sampleChat, unreadMessageCount: 0 },
      { ...otherChat, unreadMessageCount: 0 },
    ]);
    vi.mocked(fetchMessages).mockResolvedValue([sampleMessage]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(screen.getByText("hello")).toBeInTheDocument();
    });

    // sampleChat is selected (first chat) and the tab is visible by
    // default in jsdom, so only otherChat's increase should notify.
    await act(async () => {
      onChatUnread?.({
        type: "chat.unread",
        chats: [
          { chatId: sampleChat.id, unreadMessageCount: 1, updatedAt: "2026-06-11T12:05:00.000Z" },
          { chatId: otherChat.id, unreadMessageCount: 1, updatedAt: "2026-06-11T12:05:00.000Z" },
        ],
      });
    });

    expect(notifyNewMessage).toHaveBeenCalledTimes(1);
    expect(notifyNewMessage).toHaveBeenCalledWith(
      expect.objectContaining({ id: otherChat.id }),
    );
  });

  it("selects the chat referenced by a notification click event", async () => {
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([sampleChat, otherChat]);
    vi.mocked(fetchMessages).mockResolvedValue([]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(fetchMessages).toHaveBeenCalledWith(sampleChat.id);
    });

    await act(async () => {
      window.dispatchEvent(
        new CustomEvent(NOTIFICATION_CHAT_SELECT_EVENT, {
          detail: { chatId: otherChat.id },
        }),
      );
    });

    await waitFor(() => {
      expect(fetchMessages).toHaveBeenCalledWith(otherChat.id);
    });
  });

  it("selects a chat left pending by a notification click from before this tab existed", async () => {
    vi.mocked(consumePendingNotificationChatId).mockReturnValue(otherChat.id);
    vi.mocked(useAuth).mockReturnValue({
      user: authUser,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });
    vi.mocked(fetchChats).mockResolvedValue([sampleChat, otherChat]);
    vi.mocked(fetchMessages).mockResolvedValue([]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(fetchMessages).toHaveBeenCalledWith(otherChat.id);
    });
  });
});
