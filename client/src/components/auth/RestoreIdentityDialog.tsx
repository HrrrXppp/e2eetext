import { FormEvent, useEffect, useId, useRef, useState } from "react";
import { importIdentityBackup } from "@/lib/e2ee/crypto";
import { fetchIdentityPublicKey, saveStoredIdentity, uploadIdentityKey } from "@/lib/e2ee/storage";
import type { StoredIdentity } from "@/lib/e2ee/types";

type RestoreIdentityDialogProps = {
  userId: string;
  onClose: () => void;
};

// File.text() is not implemented everywhere this app needs to run (some
// browsers/environments only support the older FileReader API), so read
// via FileReader for broader compatibility.
function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ""));
    reader.onerror = () => reject(reader.error ?? new Error("could not read file"));
    reader.readAsText(file);
  });
}

export function RestoreIdentityDialog({ userId, onClose }: RestoreIdentityDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [passphrase, setPassphrase] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingIdentity, setPendingIdentity] = useState<StoredIdentity | null>(null);
  const [done, setDone] = useState(false);

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

  async function finishImport(identity: StoredIdentity) {
    await saveStoredIdentity(userId, identity);
    try {
      await uploadIdentityKey(userId, identity);
    } catch {
      // Local restore already succeeded; the next sign-in retries the
      // server upload (see ensureIdentityKeys in session.ts), so a failed
      // upload here is not fatal to the restore itself.
    }
    setDone(true);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    if (!file) {
      setError("Choose a backup file.");
      return;
    }
    if (!passphrase) {
      setError("Enter the backup passphrase.");
      return;
    }

    setSubmitting(true);
    try {
      const raw = await readFileAsText(file);
      const identity = await importIdentityBackup(raw, passphrase);

      // Multi-device / re-import collision check: this device (or account)
      // may already have a different identity uploaded. Overwriting it
      // silently would invalidate existing chat-key wraps other
      // participants hold for the current key, so confirm explicitly first.
      let serverKey: string | null = null;
      try {
        serverKey = (await fetchIdentityPublicKey(userId)).publicKey;
      } catch {
        serverKey = null;
      }

      if (serverKey && serverKey !== identity.publicKey) {
        setPendingIdentity(identity);
        return;
      }

      await finishImport(identity);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not restore backup. Try again.");
    } finally {
      // Best-effort memory hygiene: clear the passphrase input regardless
      // of outcome.
      setPassphrase("");
      setSubmitting(false);
    }
  }

  async function handleConfirmOverwrite() {
    if (!pendingIdentity) {
      return;
    }
    setSubmitting(true);
    try {
      await finishImport(pendingIdentity);
    } catch {
      setError("Could not restore backup. Try again.");
    } finally {
      setSubmitting(false);
      setPendingIdentity(null);
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
          aria-label="Close restore dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Restore your private key
        </h2>
        <p className="sign-in-dialog__lead">
          Restoring replaces the private key stored on this device with the one from your backup file.
          This recovers your identity after losing access to your original device -- it is not a way to
          keep multiple devices in sync.
        </p>

        {done ? (
          <>
            <p className="new-chat-dialog__hint">Backup restored on this device.</p>
            <button type="button" className="new-chat-dialog__submit" onClick={onClose}>
              Done
            </button>
          </>
        ) : pendingIdentity ? (
          <>
            <p className="new-chat-dialog__error" role="alert">
              This account already has a different key on file. Restoring will replace it and update
              the key your account presents to other participants -- anyone you chat with may need to
              re-share existing conversations. Continue?
            </p>
            <button
              type="button"
              className="new-chat-dialog__submit"
              disabled={submitting}
              onClick={handleConfirmOverwrite}
            >
              {submitting ? "Restoring..." : "Overwrite and restore"}
            </button>
            <button type="button" className="new-chat-dialog__back" onClick={() => setPendingIdentity(null)}>
              Cancel
            </button>
          </>
        ) : (
          <form className="new-chat-dialog__form" onSubmit={handleSubmit}>
            <label className="new-chat-dialog__field">
              <span>Backup file</span>
              <input
                type="file"
                accept="application/json,.json"
                onChange={(event) => setFile(event.target.files?.[0] ?? null)}
              />
            </label>
            <label className="new-chat-dialog__field">
              <span>Backup passphrase</span>
              <input
                type="password"
                value={passphrase}
                onChange={(event) => setPassphrase(event.target.value)}
                placeholder="Passphrase used at export"
                autoComplete="current-password"
              />
            </label>

            {error ? (
              <p className="new-chat-dialog__error" role="alert">
                {error}
              </p>
            ) : null}

            <button type="submit" className="new-chat-dialog__submit" disabled={submitting}>
              {submitting ? "Restoring..." : "Restore"}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
