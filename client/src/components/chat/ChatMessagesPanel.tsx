import { FormEvent, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useMarkMessageReadOnView } from "@/hooks/useMarkMessageReadOnView";
import { formatChatUpdatedAt, type Chat } from "@/lib/chats";
import { formatDisappearAt, messageExpiresAt } from "@/lib/disappear";
import type { Message } from "@/lib/messages";
import { userLabel } from "@/lib/users";

type ChatMessagesPanelProps = {
  chat: Chat | null;
  messages: Message[];
  currentUserId: string;
  loading: boolean;
  error: string | null;
  sending: boolean;
  sendError: string | null;
  onSend: (data: string) => Promise<void>;
  onMarkRead: (messageId: string) => void;
};

function chatLabel(chat: Chat): string {
  return chat.name?.trim() || "Unnamed chat";
}

function formatMessageTime(value: string): string {
  return new Date(value).toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}

function firstUnreadIndex(messages: Message[]): number {
  return messages.findIndex((message) => message.unread);
}

export function ChatMessagesPanel({
  chat,
  messages,
  currentUserId,
  loading,
  error,
  sending,
  sendError,
  onSend,
  onMarkRead,
}: ChatMessagesPanelProps) {
  const [draft, setDraft] = useState("");
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const unreadDividerRef = useRef<HTMLDivElement>(null);
  const scrolledToUnreadForChatRef = useRef<string | null>(null);
  const unreadDividerIndex = useMemo(() => firstUnreadIndex(messages), [messages]);
  const setMessageRef = useMarkMessageReadOnView(
    chat?.id ?? null,
    messages,
    currentUserId,
    messagesContainerRef,
    onMarkRead,
  );

  useLayoutEffect(() => {
    if (!chat || loading || error) {
      return;
    }

    if (scrolledToUnreadForChatRef.current === chat.id) {
      return;
    }

    if (unreadDividerIndex <= 0) {
      scrolledToUnreadForChatRef.current = chat.id;
      return;
    }

    unreadDividerRef.current?.scrollIntoView({ block: "start" });
    scrolledToUnreadForChatRef.current = chat.id;
  }, [chat, loading, error, unreadDividerIndex]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = draft.trim();
    if (!data || sending) {
      return;
    }

    await onSend(data);
    setDraft("");
  }

  if (!chat) {
    return (
      <section className="chats-page__panel chats-page__panel--empty">
        <p className="chats-page__status">Select a chat to view messages.</p>
      </section>
    );
  }

  return (
    <section className="chats-page__panel" aria-label={`Messages in ${chatLabel(chat)}`}>
      <header className="chats-page__panel-header">
        <div>
          <h2 className="chats-page__panel-title">{chatLabel(chat)}</h2>
          <p className="chats-page__panel-meta">
            Updated{" "}
            <time dateTime={chat.updatedAt}>{formatChatUpdatedAt(chat.updatedAt)}</time>
            {" · "}
            Messages disappear after {Math.round(chat.disappearAfterMinutes / (24 * 60))}d
          </p>
        </div>
      </header>

      <div className="chats-page__messages" ref={messagesContainerRef}>
        {loading ? (
          <p className="chats-page__status">Loading messages...</p>
        ) : error ? (
          <p className="chats-page__status chats-page__status--error" role="alert">
            {error}
          </p>
        ) : messages.length === 0 ? (
          <p className="chats-page__status">No messages yet. Say hello.</p>
        ) : (
          <ul className="chats-page__message-list">
            {messages.map((message, index) => {
              const own = message.userId === currentUserId;
              const showUnreadDivider = unreadDividerIndex > 0 && index === unreadDividerIndex;

              return (
                <li key={message.id} className="chats-page__message-item">
                  {showUnreadDivider ? (
                    <div
                      ref={unreadDividerRef}
                      className="chats-page__message-divider"
                      role="separator"
                      aria-label="Unread messages"
                    >
                      <span>Unread messages</span>
                    </div>
                  ) : null}
                  <article
                    ref={(element) => setMessageRef(message.id, element)}
                    data-message-id={message.id}
                    className={`chats-page__message${own ? " chats-page__message--own" : ""}${
                      message.unread ? " chats-page__message--unread" : ""
                    }`}
                  >
                    <p className="chats-page__message-author">
                      {userLabel(message.userName, message.userId)}
                    </p>
                    <p className="chats-page__message-text">{message.data}</p>
                    <p className="chats-page__message-time">
                      <time dateTime={message.createdAt}>{formatMessageTime(message.createdAt)}</time>
                      {" · disappears "}
                      <time
                        dateTime={messageExpiresAt(
                          message.createdAt,
                          chat.disappearAfterMinutes,
                        ).toISOString()}
                      >
                        {formatDisappearAt(message.createdAt, chat.disappearAfterMinutes)}
                      </time>
                    </p>
                  </article>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <form className="chats-page__composer" onSubmit={handleSubmit}>
        <label className="chats-page__composer-label" htmlFor="message-draft">
          Message
        </label>
        <div className="chats-page__composer-row">
          <textarea
            id="message-draft"
            className="chats-page__composer-input"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="Write a message..."
            rows={2}
            disabled={sending}
          />
          <button
            type="submit"
            className="chats-page__composer-send"
            disabled={sending || !draft.trim()}
          >
            {sending ? "Sending..." : "Send"}
          </button>
        </div>
        {sendError ? (
          <p className="chats-page__status chats-page__status--error" role="alert">
            {sendError}
          </p>
        ) : null}
      </form>
    </section>
  );
}
