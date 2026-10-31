import { useCallback, useEffect, useId, useRef, useState } from "react";
import { NewChatMemberSearch } from "@/components/chat/NewChatMemberSearch";
import { addMembers, getChatMembers, removeMember, type Chat, type ChatMember } from "@/lib/chats";
import { prepareAddMembers } from "@/lib/e2ee/session";
import { userLabel, type User } from "@/lib/users";

type ChatSettingsDialogProps = {
  chat: Chat;
  currentUserId: string;
  onClose: () => void;
};

export function ChatSettingsDialog({ chat, currentUserId, onClose }: ChatSettingsDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [members, setMembers] = useState<ChatMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [removingId, setRemovingId] = useState<string | null>(null);
  const [view, setView] = useState<"members" | "search">("members");
  const [pending, setPending] = useState<User[]>([]);
  const [adding, setAdding] = useState(false);
  const isAdmin = Boolean(chat.isAdmin);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") {
        return;
      }
      if (view === "search") {
        setView("members");
        return;
      }
      onClose();
    }

    document.addEventListener("keydown", handleKeyDown);
    dialogRef.current?.focus();

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [onClose, view]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(null);

    getChatMembers(chat.id)
      .then((items) => {
        if (active) {
          setMembers(items);
        }
      })
      .catch(() => {
        if (active) {
          setError("Could not load chat members. Try again later.");
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [chat.id]);

  const handleRemove = useCallback(
    async (member: ChatMember) => {
      const label = userLabel(member.name, member.id);
      if (!window.confirm(`Remove ${label} from this chat?`)) {
        return;
      }

      setRemovingId(member.id);
      setError(null);
      try {
        await removeMember(chat.id, member.id);
        setMembers((current) => current.filter((item) => item.id !== member.id));
      } catch {
        setError("Could not remove member. Try again.");
      } finally {
        setRemovingId(null);
      }
    },
    [chat.id],
  );

  const handleSelect = useCallback((user: User) => {
    setPending((current) => (current.some((item) => item.id === user.id) ? current : [...current, user]));
    setView("members");
  }, []);

  const handleConfirmAdd = useCallback(async () => {
    setAdding(true);
    setError(null);
    try {
      const userIds = pending.map((user) => user.id);
      const prepared = await prepareAddMembers(currentUserId, chat.id, userIds);
      await addMembers(chat.id, userIds, prepared.keyId, prepared.wraps);
      setPending([]);
      setMembers(await getChatMembers(chat.id));
    } catch (err) {
      setError(addMembersErrorMessage(err));
    } finally {
      setAdding(false);
    }
  }, [chat.id, currentUserId, pending]);

  return (
    <div className="sign-in-dialog__backdrop" onClick={onClose}>
      <div
        ref={dialogRef}
        className="sign-in-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        onClick={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          className="sign-in-dialog__close"
          aria-label="Close chat settings dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Chat Settings
        </h2>
        <p className="sign-in-dialog__lead">
          {isAdmin ? "Members of this chat. You can remove other members." : "Members of this chat."}
        </p>

        {view === "search" ? (
          <NewChatMemberSearch
            currentUserId={currentUserId}
            members={[...members, ...pending]}
            onAddMember={handleSelect}
            onBack={() => setView("members")}
          />
        ) : null}

        {view === "search" ? null : error ? (
          <p className="new-chat-dialog__error" role="alert">
            {error}
          </p>
        ) : null}

        {view === "search" ? null : loading ? (
          <p className="new-chat-dialog__status">Loading members...</p>
        ) : (
          <ul className="chat-settings__members" aria-label="Chat members">
            {members.map((member) => {
              const self = member.id === currentUserId;
              return (
                <li key={member.id} className="chat-settings__member">
                  <span className="chat-settings__name">
                    <span className="chat-settings__label" title={userLabel(member.name, member.id)}>
                      {userLabel(member.name, member.id)}
                    </span>
                    {self ? <span className="chat-settings__badge">you</span> : null}
                    {member.isAdmin ? <span className="chat-settings__badge">admin</span> : null}
                  </span>
                  {isAdmin && !self ? (
                    <button
                      type="button"
                      className="new-chat-dialog__back"
                      disabled={removingId === member.id}
                      aria-label={`Remove ${userLabel(member.name, member.id)}`}
                      onClick={() => void handleRemove(member)}
                    >
                      {removingId === member.id ? "Removing..." : "Remove"}
                    </button>
                  ) : null}
                </li>
              );
            })}
          </ul>
        )}

        {view === "members" && isAdmin && !loading ? (
          <div className="chat-settings__add">
            {pending.length > 0 ? (
              <ul className="new-chat-dialog__member-list" aria-label="Members to add">
                {pending.map((user) => (
                  <li key={user.id} className="new-chat-dialog__member">
                    <span>{userLabel(user.name, user.id)}</span>
                    <button
                      type="button"
                      className="new-chat-dialog__remove"
                      aria-label={`Do not add ${userLabel(user.name, user.id)}`}
                      onClick={() => setPending((current) => current.filter((item) => item.id !== user.id))}
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
            ) : null}
            <button
              type="button"
              className="new-chat-dialog__submit"
              disabled={adding}
              onClick={() => setView("search")}
            >
              Add members
            </button>
            {pending.length > 0 ? (
              <button
                type="button"
                className="new-chat-dialog__submit"
                disabled={adding}
                onClick={() => void handleConfirmAdd()}
              >
                {adding ? "Adding..." : `Add ${pending.length} ${pending.length === 1 ? "member" : "members"}`}
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function addMembersErrorMessage(err: unknown): string {
  const message = err instanceof Error ? err.message : "";
  if (message === "chat admin required") {
    return "Only chat admins can add members.";
  }
  if (message.startsWith("already a member")) {
    return `Could not add members: ${message}.`;
  }
  if (message.startsWith("identity key not found for ")) {
    return `Could not add members: ${message}. They need to sign in once first.`;
  }
  return "Could not add members. Try again.";
}
