import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChatMessagesPanel } from "@/components/chat/ChatMessagesPanel";
import { ChatSidebar } from "@/components/chat/ChatSidebar";
import { NewChatDialog } from "@/components/chat/NewChatDialog";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { useAuth } from "@/hooks/useAuth";
import { useChatSocket } from "@/hooks/useChatSocket";
import { fetchChats, type Chat } from "@/lib/chats";
import { decryptIncomingMessage, encryptOutgoingMessage } from "@/lib/e2ee/session";
import { createMessage, fetchMessages, markMessageRead, type Message } from "@/lib/messages";
import { filterActiveMessages } from "@/lib/disappear";
import type { ChatAddedEvent, ChatUnreadEvent } from "@/lib/ws";

export function ChatsPage() {
  const { user, loading: authLoading } = useAuth();
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
  const messagesRef = useRef(messages);
  messagesRef.current = messages;

  const selectedChat = useMemo(
    () => chats.find((chat) => chat.id === selectedChatId) ?? null,
    [chats, selectedChatId],
  );

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
    } catch {
      setChatsError("Could not load your chats. Try again later.");
    } finally {
      if (!options?.silent) {
        setChatsLoading(false);
      }
    }
  }, []);

  const loadMessages = useCallback(
    async (chatId: string, viewerUserId: string) => {
      setMessagesLoading(true);
      setMessagesError(null);

      try {
        const items = await fetchMessages(chatId);
        const decrypted = await Promise.all(
          items.map(async (message) => {
            try {
              return {
                ...message,
                data: await decryptIncomingMessage(message, viewerUserId),
              };
            } catch {
              return { ...message, data: "Error decrypt message" };
            }
          }),
        );
        setMessages(decrypted);
      } catch {
        setMessagesError("Could not load messages. Try again later.");
        setMessages([]);
      } finally {
        setMessagesLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    if (!selectedChat) {
      return;
    }
    const ttl = selectedChat.disappearAfterMinutes;
    const tick = () => {
      setMessages((current) => filterActiveMessages(current, ttl));
    };
    tick();
    const id = window.setInterval(tick, 30_000);
    return () => window.clearInterval(id);
  }, [selectedChat]);

  useEffect(() => {
    if (authLoading) {
      return;
    }

    if (!user) {
      setChatsLoading(false);
      return;
    }

    void loadChats(user.id);
  }, [authLoading, user, loadChats]);

  useEffect(() => {
    if (!selectedChatId || !user?.id) {
      setMessages([]);
      return;
    }

    void loadMessages(selectedChatId, user.id);
  }, [selectedChatId, user?.id, loadMessages]);

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

        if (incomingCount !== previousCount && user?.id) {
          void loadMessages(selectedChatId, user.id);
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
    [chats, loadMessages, selectedChatId, user?.id],
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
        if (user?.id) {
          void loadMessages(selectedChatId, user.id);
          void loadChats(user.id);
        }
      });
    },
    [loadChats, loadMessages, selectedChatId, user?.id],
  );

  async function handleSendMessage(data: string) {
    if (!user || !selectedChatId) {
      return;
    }

    setSending(true);
    setSendError(null);

    try {
      const encrypted = await encryptOutgoingMessage({
        plaintext: data,
        chatId: selectedChatId,
        senderUserId: user.id,
      });
      const message = await createMessage({
        chatId: selectedChatId,
        userId: user.id,
        data: encrypted,
      });
      setMessages((current) => [...current, { ...message, data }]);
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
    } catch (error) {
      console.error("send message failed", error);
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

  const canManageChats = Boolean(user) && !authLoading;
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
          onClose={() => setCreateOpen(false)}
          onCreated={handleChatCreated}
        />
      ) : null}
    </>
  );
}
