import { FormEvent, useEffect, useId, useRef, useState } from "react";
import { NewChatMemberSearch } from "@/components/chat/NewChatMemberSearch";
import { createChat, type Chat } from "@/lib/chats";
import { DEFAULT_DISAPPEAR_AFTER_MINUTES } from "@/lib/disappear";
import { prepareE2EEChatCreation, rememberCreatedChatKey } from "@/lib/e2ee/session";
import { userLabel, type User } from "@/lib/users";

type NewChatDialogProps = {
  currentUserId: string;
  onClose: () => void;
  onCreated: (chat: Chat) => void;
};

type DialogView = "form" | "search";

export function NewChatDialog({ currentUserId, onClose, onCreated }: NewChatDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<DialogView>("form");
  const [name, setName] = useState("");
  const [members, setMembers] = useState<User[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") {
        return;
      }

      if (view === "search") {
        setView("form");
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

  function handleRemoveMember(userId: string) {
    setMembers((current) => current.filter((member) => member.id !== userId));
  }

  function handleAddMember(member: User) {
    setMembers((current) => {
      if (current.some((item) => item.id === member.id)) {
        return current;
      }
      return [...current, member];
    });
    setView("form");
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const trimmedName = name.trim();
    if (!trimmedName) {
      setError("Chat name is required.");
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const memberIds = [currentUserId, ...members.map((member) => member.id)];
      const prepared = await prepareE2EEChatCreation({
        currentUserId,
        memberUserIds: memberIds,
      });
      const chat = await createChat({
        name: trimmedName,
        usersUids: memberIds,
        keyId: prepared.keyId,
        wraps: prepared.wraps,
        disappearAfterMinutes: DEFAULT_DISAPPEAR_AFTER_MINUTES,
      });
      await rememberCreatedChatKey(currentUserId, chat.id, prepared.keyId, prepared.chatKey);
      onCreated(chat);
      onClose();
    } catch {
      setError("Could not create chat. Try again.");
    } finally {
      setSubmitting(false);
    }
  }

  const title = view === "search" ? "Search users" : "New chat";
  const lead =
    view === "search"
      ? "Find people to add to this chat."
      : "Create a conversation and add members.";
  const disappearDays = DEFAULT_DISAPPEAR_AFTER_MINUTES / (24 * 60);

  return (
    <div className="new-chat-dialog__backdrop" onClick={onClose}>
      <div
        ref={dialogRef}
        className="new-chat-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        onClick={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          className="new-chat-dialog__close"
          aria-label="Close new chat dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="new-chat-dialog__title">
          {title}
        </h2>
        <p className="new-chat-dialog__lead">{lead}</p>

        {view === "search" ? (
          <NewChatMemberSearch
            currentUserId={currentUserId}
            members={members}
            onAddMember={handleAddMember}
            onBack={() => setView("form")}
          />
        ) : (
          <form className="new-chat-dialog__form" onSubmit={handleSubmit}>
            <label className="new-chat-dialog__field">
              <span>Chat name</span>
              <input
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Project team"
                autoComplete="off"
              />
            </label>

            <p className="new-chat-dialog__hint">
              Messages in this chat disappear after {disappearDays} days.
            </p>

            <div className="new-chat-dialog__members">
              <span className="new-chat-dialog__members-label">Members</span>
              <ul className="new-chat-dialog__member-list">
                <li className="new-chat-dialog__member new-chat-dialog__member--self">You</li>
                {members.map((member) => (
                  <li key={member.id} className="new-chat-dialog__member">
                    <span>{userLabel(member.name, member.id)}</span>
                    <button
                      type="button"
                      className="new-chat-dialog__remove"
                      aria-label={`Remove ${userLabel(member.name, member.id)}`}
                      onClick={() => handleRemoveMember(member.id)}
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
            </div>

            <button type="button" className="new-chat-dialog__submit" onClick={() => setView("search")}>
              Search users
            </button>

            {error ? (
              <p className="new-chat-dialog__error" role="alert">
                {error}
              </p>
            ) : null}

            <button type="submit" className="new-chat-dialog__submit" disabled={submitting}>
              {submitting ? "Creating..." : "Create chat"}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
