import "fake-indexeddb/auto";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChatsPage } from "@/app/ChatsPage";
import { encodeBase64 } from "@/lib/bytes";
import { encryptMessage, generateKemKeyPair, wrapChatPrivateKeyForMember } from "@/lib/crypto";
import { saveOwnKeyPair } from "@/lib/keyStore";

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

vi.mock("@/lib/messages", () => ({
  fetchMessages: vi.fn(),
  createMessage: vi.fn(),
}));

vi.mock("@/hooks/useChatSocket", () => ({
  useChatSocket: vi.fn(),
}));

import { useAuth } from "@/hooks/useAuth";
import { useChatSocket } from "@/hooks/useChatSocket";
import { fetchChats } from "@/lib/chats";
import { createMessage, fetchMessages } from "@/lib/messages";
import type { ChatUnreadEvent } from "@/lib/ws";

// These tests exercise real ML-KEM crypto + IndexedDB (no mocking), which
// can be slow under heavy parallel test-file load; the default 1000ms
// waitFor timeout (and vitest's default 5000ms per-test timeout) are too
// tight for that under contention.
const WAIT_OPTS = { timeout: 15000 };
const TEST_TIMEOUT = 20000;

let onChatUnread: ((event: ChatUnreadEvent) => void) | undefined;

vi.mocked(useChatSocket).mockImplementation((_enabled, callback) => {
  onChatUnread = callback;
  return "open";
});

const ownKeyPair = generateKemKeyPair();
const chatKeyPair = generateKemKeyPair();

const authUser = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  subject: "google-subject-1",
  provider: "google",
  oidcProviderId: "11111111-1111-1111-1111-111111111111",
  kemPublicKey: encodeBase64(ownKeyPair.publicKey),
};

await saveOwnKeyPair(authUser.id, ownKeyPair);

const { kemCiphertext: chatKemCiphertext, wrappedChatPrivateKey } = await wrapChatPrivateKeyForMember(
  chatKeyPair.secretKey,
  ownKeyPair.publicKey,
);

const sampleChat = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "General",
  adminUserId: authUser.id,
  kemPublicKey: encodeBase64(chatKeyPair.publicKey),
  wrappedChatPrivateKey: encodeBase64(wrappedChatPrivateKey),
  kemCiphertext: encodeBase64(chatKemCiphertext),
  createdAt: "2026-06-11T12:00:00.000Z",
  updatedAt: "2026-06-11T12:00:00.000Z",
};

async function encryptFixture(plaintext: string) {
  const { kemCiphertext, data } = await encryptMessage(chatKeyPair.publicKey, plaintext);
  return { data: encodeBase64(data), kemCiphertext: encodeBase64(kemCiphertext) };
}

const helloEnvelope = await encryptFixture("hello");

const sampleMessage = {
  id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  chatId: sampleChat.id,
  userId: authUser.id,
  data: helloEnvelope.data,
  kemCiphertext: helloEnvelope.kemCiphertext,
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
    }, WAIT_OPTS);

    await waitFor(() => {
      expect(screen.getByText("hello", { selector: ".chats-page__message-text" })).toBeInTheDocument();
    }, WAIT_OPTS);
    expect(fetchMessages).toHaveBeenCalledWith(sampleChat.id);
    expect(screen.getByLabelText("Message")).toBeInTheDocument();
  }, TEST_TIMEOUT);

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

    // Wait for the initial (empty) message load to finish before sending,
    // so it can't race with and clobber the optimistically-appended send.
    await waitFor(() => {
      expect(screen.getByText("No messages yet. Say hello.")).toBeInTheDocument();
    }, WAIT_OPTS);

    fireEvent.change(screen.getByLabelText("Message"), {
      target: { value: "hello" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    // Wait for the message to actually render (i.e. state has settled) before
    // inspecting the call args, rather than relying on mock.calls being
    // populated at invocation time to imply the resulting state update ran.
    // Scoped to the message bubble specifically — the composer textarea also
    // contains the literal text "hello" (the still-unsubmitted draft echo)
    // until onSend resolves, which would otherwise make this query ambiguous
    // and resolve before the send actually completes.
    await waitFor(() => {
      expect(screen.getByText("hello", { selector: ".chats-page__message-text" })).toBeInTheDocument();
    }, WAIT_OPTS);

    expect(createMessage).toHaveBeenCalledTimes(1);
    const [input] = vi.mocked(createMessage).mock.calls[0];
    expect(input.chatId).toBe(sampleChat.id);
    expect(input.userId).toBe(authUser.id);
    expect(typeof input.data).toBe("string");
    expect(input.data.length).toBeGreaterThan(0);
    expect(typeof input.kemCiphertext).toBe("string");
    expect(input.kemCiphertext.length).toBeGreaterThan(0);
  }, TEST_TIMEOUT);

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
    }, WAIT_OPTS);

    expect(screen.getByRole("button", { name: "New chat" })).toBeInTheDocument();
    expect(screen.getByText("Select a chat to view messages.")).toBeInTheDocument();
  }, TEST_TIMEOUT);

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
    vi.mocked(fetchChats).mockResolvedValue([{ ...sampleChat, unreadMessageCount: 0 }]);

    const newMessageEnvelope = await encryptFixture("new message");
    vi.mocked(fetchMessages)
      .mockResolvedValueOnce([sampleMessage])
      .mockResolvedValueOnce([
        sampleMessage,
        {
          ...sampleMessage,
          id: "cccccccc-cccc-cccc-cccc-cccccccccccc",
          data: newMessageEnvelope.data,
          kemCiphertext: newMessageEnvelope.kemCiphertext,
        },
      ]);

    render(<ChatsPage />);

    await waitFor(() => {
      expect(screen.getByText("hello", { selector: ".chats-page__message-text" })).toBeInTheDocument();
    }, WAIT_OPTS);
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
    }, WAIT_OPTS);
    await waitFor(() => {
      expect(screen.getByText("new message", { selector: ".chats-page__message-text" })).toBeInTheDocument();
    }, WAIT_OPTS);
  }, TEST_TIMEOUT);
});
