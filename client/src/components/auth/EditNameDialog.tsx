import { FormEvent, useEffect, useId, useRef, useState } from "react";
import { updateUserName } from "@/lib/users";

type EditNameDialogProps = {
  userId: string;
  currentName?: string;
  onClose: () => void;
  onSaved: (name?: string) => void;
};

export function EditNameDialog({ userId, currentName, onClose, onSaved }: EditNameDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [name, setName] = useState(currentName ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

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

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      const user = await updateUserName(userId, name.trim());
      onSaved(user.name);
    } catch {
      setError("Could not update name. Try again.");
    } finally {
      setSubmitting(false);
    }
  }

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
          aria-label="Close edit name dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Edit name
        </h2>
        <p className="sign-in-dialog__lead">Choose how your name appears in the app.</p>

        <form className="new-chat-dialog__form" onSubmit={handleSubmit}>
          <label className="new-chat-dialog__field">
            <span>Display name</span>
            <input
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Your name"
              autoComplete="name"
            />
          </label>
          <p className="new-chat-dialog__hint">Leave empty to show your user ID.</p>

          {error ? (
            <p className="new-chat-dialog__error" role="alert">
              {error}
            </p>
          ) : null}

          <button type="submit" className="new-chat-dialog__submit" disabled={submitting}>
            {submitting ? "Saving..." : "Save name"}
          </button>
        </form>
      </div>
    </div>
  );
}
