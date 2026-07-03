import { formatChatUpdatedAt, type Chat } from "@/lib/chats";

type ChatSidebarProps = {
  chats: Chat[];
  selectedChatId: string | null;
  loading: boolean;
  error: string | null;
  onSelect: (chatId: string) => void;
  onNewChat: () => void;
};

function chatLabel(chat: Chat): string {
  return chat.name?.trim() || "Unnamed chat";
}

export function ChatSidebar({
  chats,
  selectedChatId,
  loading,
  error,
  onSelect,
  onNewChat,
}: ChatSidebarProps) {
  return (
    <aside className="chats-page__sidebar">
      <div className="chats-page__sidebar-header">
        <h1 className="chats-page__title">Chats</h1>
        <button type="button" className="chats-page__new" onClick={onNewChat}>
          New chat
        </button>
      </div>

      {loading ? (
        <p className="chats-page__status">Loading chats...</p>
      ) : error ? (
        <p className="chats-page__status chats-page__status--error" role="alert">
          {error}
        </p>
      ) : chats.length === 0 ? (
        <p className="chats-page__status">No chats yet.</p>
      ) : (
        <ul className="chats-page__list" role="listbox" aria-label="Chats">
          {chats.map((chat) => {
            const selected = chat.id === selectedChatId;
            return (
              <li key={chat.id}>
                <button
                  type="button"
                  className={`chats-page__item${selected ? " chats-page__item--active" : ""}`}
                  role="option"
                  aria-selected={selected}
                  onClick={() => onSelect(chat.id)}
                >
                  <span className="chats-page__item-row">
                    <span className="chats-page__item-name">{chatLabel(chat)}</span>
                    {(chat.unreadMessageCount ?? 0) > 0 ? (
                      <span
                        className="chats-page__item-unread"
                        aria-label={`${chat.unreadMessageCount} unread messages`}
                      >
                        {chat.unreadMessageCount}
                      </span>
                    ) : null}
                  </span>
                  <time className="chats-page__item-date" dateTime={chat.updatedAt}>
                    {formatChatUpdatedAt(chat.updatedAt)}
                  </time>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </aside>
  );
}
