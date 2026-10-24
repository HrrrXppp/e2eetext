import { useCallback, useEffect, useId, useRef, useState } from "react";
import { getChatMembers, removeMember, type Chat, type ChatMember } from "@/lib/chats";
import { userLabel } from "@/lib/users";

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
  const isAdmin = Boolean(chat.isAdmin);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    dialogRef.current?.focus();

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [onClose]);

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

        {error ? (
          <p className="new-chat-dialog__error" role="alert">
            {error}
          </p>
        ) : null}

        {loading ? (
          <p className="new-chat-dialog__status">Loading members...</p>
        ) : (
          <ul className="chat-settings__members" aria-label="Chat members">
            {members.map((member) => {
              const self = member.id === currentUserId;
              return (
                <li key={member.id} className="chat-settings__member">
                  <span>
                    {userLabel(member.name, member.id)}
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
      </div>
    </div>
  );
}
