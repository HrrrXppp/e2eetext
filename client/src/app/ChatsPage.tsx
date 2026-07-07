import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChatMessagesPanel } from "@/components/chat/ChatMessagesPanel";
import { ChatSidebar } from "@/components/chat/ChatSidebar";
import { NewChatDialog } from "@/components/chat/NewChatDialog";
import { KeyBackupDialog } from "@/components/auth/KeyBackupDialog";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { useAuth } from "@/hooks/useAuth";
import { useChatSocket } from "@/hooks/useChatSocket";
import { decodeBase64, encodeBase64 } from "@/lib/bytes";
import { fetchChats, type Chat } from "@/lib/chats";
import { decryptMessage, encryptMessage, unwrapChatPrivateKey } from "@/lib/crypto";
import { loadChatPrivateKey, loadOwnKeyPair, saveChatPrivateKey } from "@/lib/keyStore";
import { createMessage, fetchMessages, markMessageRead, type Message } from "@/lib/messages";
import type { ChatAddedEvent, ChatUnreadEvent } from "@/lib/ws";

// Unwraps and caches a chat's private key on demand, using the chat's own
// wrap data (always present on any Chat the caller belongs to) rather than
// depending on a separate prefetch step having already completed — avoids a
// race between chat-list loading and message decryption.
async function getOrUnwrapChatPrivateKey(userId: string, chat: Chat): Promise<Uint8Array | null> {
  const cached = await loadChatPrivateKey(userId, chat.id);
  if (cached) {
    return cached;
  }

  const ownKeyPair = await loadOwnKeyPair(userId);
  if (!ownKeyPair) {
    return null;
  }

  try {
    const chatPrivateKey = await unwrapChatPrivateKey(
      decodeBase64(chat.kemCiphertext),
      decodeBase64(chat.wrappedChatPrivateKey),
      ownKeyPair.secretKey,
    );
    await saveChatPrivateKey(userId, chat.id, chatPrivateKey);
    return chatPrivateKey;
  } catch {
    return null;
  }
}

async function decryptMessages(userId: string, chat: Chat, items: Message[]): Promise<Message[]> {
  const chatPrivateKey = await getOrUnwrapChatPrivateKey(userId, chat);
  if (!chatPrivateKey) {
    return items;
  }

  return Promise.all(
    items.map(async (message) => {
      try {
        const plaintext = await decryptMessage(
          decodeBase64(message.kemCiphertext),
          decodeBase64(message.data),
          chatPrivateKey,
        );
        return { ...message, plaintext };
      } catch {
        return message;
      }
    }),
  );
}

export function ChatsPage() {
  const { user, loading: authLoading } = useAuth();
  const [ownKeyState, setOwnKeyState] = useState<"checking" | "ready" | "missing">("checking");
  const [chats, setChats] = useState<Chat[]>([]);
  const [chatsLoading, setChatsLoading] = useState(true);
  const [chatsError, setChatsError] = useState<string | null>(null);
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [messagesError, setMessagesError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [keyBackupOpen, setKeyBackupOpen] = useState(false);
  const messagesRef = useRef(messages);
  messagesRef.current = messages;
  const chatsRef = useRef(chats);
  chatsRef.current = chats;
  // Bumped on every loadMessages call and every successful send, so a slow
  // in-flight fetch (e.g. waiting on chat-key decryption) can't clobber a
  // message the user already sent locally in the meantime.
  const messagesRequestIdRef = useRef(0);

  const selectedChat = useMemo(
    () => chats.find((chat) => chat.id === selectedChatId) ?? null,
    [chats, selectedChatId],
  );

  useEffect(() => {
    let active = true;

    if (!user) {
      setOwnKeyState("checking");
      return;
    }

    void loadOwnKeyPair(user.id).then((keyPair) => {
      if (active) {
        setOwnKeyState(keyPair ? "ready" : "missing");
      }
    });

    return () => {
      active = false;
    };
  }, [user]);

  const loadChats = useCallback(async (userId: string, options?: { silent?: boolean }) => {
    if (!options?.silent) {
      setChatsLoading(true);
    }
    setChatsError(null);

    try {
      const items = await fetchChats(userId);
      setChats(items);
      setSelectedChatId((current) => {
        if (current && items.some((chat) => chat.id === current)) {
          return current;
        }
        return items[0]?.id ?? null;
      });

      // Warm the cache so decryption doesn't wait on this later; not required
      // for correctness since loadMessages unwraps on demand too.
      void Promise.all(items.map((chat) => getOrUnwrapChatPrivateKey(userId, chat)));
    } catch {
      setChatsError("Could not load your chats. Try again later.");
    } finally {
      if (!options?.silent) {
        setChatsLoading(false);
      }
    }
  }, []);

  const loadMessages = useCallback(
    async (chatId: string) => {
      const requestId = ++messagesRequestIdRef.current;
      setMessagesLoading(true);
      setMessagesError(null);

      try {
        const items = await fetchMessages(chatId);
        const chat = chatsRef.current.find((candidate) => candidate.id === chatId);
        const decrypted = user && chat ? await decryptMessages(user.id, chat, items) : items;
        if (requestId === messagesRequestIdRef.current) {
          setMessages(decrypted);
        }
      } catch {
        if (requestId === messagesRequestIdRef.current) {
          setMessagesError("Could not load messages. Try again later.");
          setMessages([]);
        }
      } finally {
        if (requestId === messagesRequestIdRef.current) {
          setMessagesLoading(false);
        }
      }
    },
    [user],
  );

  useEffect(() => {
    if (authLoading || ownKeyState !== "ready") {
      return;
    }

    if (!user) {
      setChatsLoading(false);
      return;
    }

    void loadChats(user.id);
  }, [authLoading, ownKeyState, user, loadChats]);

  useEffect(() => {
    if (!selectedChatId) {
      setMessages([]);
      return;
    }

    void loadMessages(selectedChatId);
  }, [selectedChatId, loadMessages]);

  function handleChatCreated(chat: Chat) {
    setChats((current) => [chat, ...current]);
    setSelectedChatId(chat.id);
    setMessages([]);
  }

  function handleOpenNewChat() {
    setCreateOpen(true);
  }

  const handleChatUnread = useCallback(
    (event: ChatUnreadEvent) => {
      const unreadByChatId = new Map(event.chats.map((chat) => [chat.chatId, chat]));

      if (selectedChatId) {
        const incoming = unreadByChatId.get(selectedChatId);
        const incomingCount = incoming?.unreadMessageCount ?? 0;
        const openChat = chats.find((chat) => chat.id === selectedChatId);
        const previousCount = openChat?.unreadMessageCount ?? 0;

        if (incomingCount !== previousCount) {
          void loadMessages(selectedChatId);
        }
      }

      setChats((current) => {
        const next = current.map((chat) => {
          const unread = unreadByChatId.get(chat.id);
          if (!unread) {
            return { ...chat, unreadMessageCount: 0 };
          }

          return {
            ...chat,
            unreadMessageCount:
              selectedChatId === chat.id ? 0 : unread.unreadMessageCount,
            updatedAt: unread.updatedAt,
          };
        });

        return [...next].sort(
          (left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime(),
        );
      });
    },
    [chats, loadMessages, selectedChatId],
  );

  const handleMarkMessageRead = useCallback(
    (messageId: string) => {
      if (!selectedChatId) {
        return;
      }

      const target = messagesRef.current.find((message) => message.id === messageId);
      if (!target?.unread) {
        return;
      }

      setMessages((current) =>
        current.map((message) =>
          message.id === messageId ? { ...message, unread: false } : message,
        ),
      );

      setChats((current) =>
        current.map((chat) =>
          chat.id === selectedChatId
            ? {
                ...chat,
                unreadMessageCount: Math.max(0, (chat.unreadMessageCount ?? 0) - 1),
              }
            : chat,
        ),
      );

      void markMessageRead(messageId).catch(() => {
        void loadMessages(selectedChatId);
        if (user?.id) {
          void loadChats(user.id);
        }
      });
    },
    [loadChats, loadMessages, selectedChatId, user?.id],
  );

  async function handleSendMessage(data: string) {
    if (!user || !selectedChatId || !selectedChat) {
      return;
    }

    setSending(true);
    setSendError(null);

    try {
      const { kemCiphertext, data: ciphertext } = await encryptMessage(
        decodeBase64(selectedChat.kemPublicKey),
        data,
      );
      const message = await createMessage({
        chatId: selectedChatId,
        userId: user.id,
        data: encodeBase64(ciphertext),
        kemCiphertext: encodeBase64(kemCiphertext),
      });
      setMessages((current) => [...current, { ...message, plaintext: data }]);
      setChats((current) =>
        [...current]
          .map((chat) =>
            chat.id === selectedChatId
              ? {
                  ...chat,
                  updatedAt: message.createdAt,
                  unreadMessageCount: 0,
                }
              : chat,
          )
          .sort(
            (left, right) =>
              new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime(),
          ),
      );
    } catch {
      setSendError("Could not send message. Try again.");
    } finally {
      setSending(false);
    }
  }

  const handleChatAdded = useCallback(
    (_event: ChatAddedEvent) => {
      if (!user?.id) {
        return;
      }
      void loadChats(user.id, { silent: true });
    },
    [loadChats, user?.id],
  );

  const canManageChats = Boolean(user) && !authLoading && ownKeyState === "ready";
  useChatSocket(canManageChats, handleChatUnread, handleChatAdded);

  return (
    <>
      <div className="chats-shell">
        <SiteHeader />
        <main className="chats-page">
          {authLoading ? (
            <p className="chats-page__status chats-page__status--centered">Loading...</p>
          ) : !user ? (
            <p className="chats-page__status chats-page__status--centered">
              Sign in to view your chats.
            </p>
          ) : ownKeyState === "checking" ? (
            <p className="chats-page__status chats-page__status--centered">Loading...</p>
          ) : ownKeyState === "missing" ? (
            <div className="chats-page__status chats-page__status--centered">
              <p>
                This device doesn&rsquo;t have your encryption private key. Without it, your
                chats can&rsquo;t be decrypted here &mdash; restore it from a backup to continue.
              </p>
              <button
                type="button"
                className="site-head__sign-in"
                onClick={() => setKeyBackupOpen(true)}
              >
                Restore from backup
              </button>
            </div>
          ) : (
            <div className="chats-page__layout">
              <ChatSidebar
                chats={chats}
                selectedChatId={selectedChatId}
                loading={chatsLoading}
                error={chatsError}
                onSelect={setSelectedChatId}
                onNewChat={handleOpenNewChat}
              />
              <ChatMessagesPanel
                chat={selectedChat}
                messages={messages}
                currentUserId={user.id}
                loading={messagesLoading}
                error={messagesError}
                sending={sending}
                sendError={sendError}
                onSend={handleSendMessage}
                onMarkRead={handleMarkMessageRead}
              />
            </div>
          )}
        </main>
      </div>

      {createOpen && canManageChats && user ? (
        <NewChatDialog
          currentUserId={user.id}
          currentUserKemPublicKey={user.kemPublicKey}
          onClose={() => setCreateOpen(false)}
          onCreated={handleChatCreated}
        />
      ) : null}

      {keyBackupOpen && user ? (
        <KeyBackupDialog
          userId={user.id}
          onClose={() => setKeyBackupOpen(false)}
          onRestored={() => setOwnKeyState("ready")}
        />
      ) : null}
    </>
  );
}
