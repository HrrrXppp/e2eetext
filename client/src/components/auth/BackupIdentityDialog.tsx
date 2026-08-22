import { FormEvent, useEffect, useId, useRef, useState } from "react";
import { exportStoredIdentityBackup } from "@/lib/e2ee/storage";

type BackupIdentityDialogProps = {
  userId: string;
  onClose: () => void;
};

type PassphraseStrength = {
  label: string;
  className: string;
};

// Visual-only signal, never blocking -- beyond requiring a non-empty passphrase, the export
// passphrase has no minimum length or complexity requirement, by design (issue #52 review feedback).
function passphraseStrength(passphrase: string): PassphraseStrength {
  let score = 0;
  if (passphrase.length >= 8) score += 1;
  if (passphrase.length >= 16) score += 1;
  if (/[a-z]/.test(passphrase) && /[A-Z]/.test(passphrase)) score += 1;
  if (/[0-9]/.test(passphrase)) score += 1;
  if (/[^A-Za-z0-9]/.test(passphrase)) score += 1;

  if (score <= 1) {
    return { label: "Weak", className: "backup-dialog__strength--weak" };
  }
  if (score <= 3) {
    return { label: "Fair", className: "backup-dialog__strength--fair" };
  }
  return { label: "Strong", className: "backup-dialog__strength--strong" };
}

function downloadBackupFile(userId: string, backup: string): void {
  const blob = new Blob([backup], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  const date = new Date().toISOString().slice(0, 10);
  link.href = url;
  link.download = `e2eetext-backup-${userId.slice(0, 8)}-${date}.json`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export function BackupIdentityDialog({ userId, onClose }: BackupIdentityDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [passphrase, setPassphrase] = useState("");
  const [confirmPassphrase, setConfirmPassphrase] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
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

  const strength = passphraseStrength(passphrase);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    if (!passphrase) {
      setError("Enter a passphrase to protect the backup file.");
      return;
    }
    if (passphrase !== confirmPassphrase) {
      setError("Passphrases do not match.");
      return;
    }

    setSubmitting(true);
    try {
      const backup = await exportStoredIdentityBackup(userId, passphrase);
      downloadBackupFile(userId, backup);
      setDone(true);
    } catch {
      setError("Could not create backup. Try again.");
    } finally {
      // Best-effort memory hygiene: clear the passphrase inputs regardless
      // of outcome -- nothing after this point needs the plaintext value.
      setPassphrase("");
      setConfirmPassphrase("");
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
          aria-label="Close backup dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Back up your private key
        </h2>
        <p className="sign-in-dialog__lead">
          This key lives only on this device and the server can never recover it. A downloaded,
          passphrase-encrypted backup is the only way to restore your identity if you lose access to
          this device.
        </p>

        {done ? (
          <>
            <p className="new-chat-dialog__hint">
              Backup downloaded. Store the file and passphrase somewhere safe -- anyone with both can
              read your messages.
            </p>
            <button type="button" className="new-chat-dialog__submit" onClick={onClose}>
              Done
            </button>
          </>
        ) : (
          <form className="new-chat-dialog__form" onSubmit={handleSubmit}>
            <label className="new-chat-dialog__field">
              <span>Backup passphrase</span>
              <input
                type="password"
                value={passphrase}
                onChange={(event) => setPassphrase(event.target.value)}
                placeholder="Choose a passphrase"
                autoComplete="new-password"
              />
            </label>
            {passphrase ? (
              <p className={`backup-dialog__strength ${strength.className}`}>{strength.label}</p>
            ) : null}
            <label className="new-chat-dialog__field">
              <span>Confirm passphrase</span>
              <input
                type="password"
                value={confirmPassphrase}
                onChange={(event) => setConfirmPassphrase(event.target.value)}
                placeholder="Re-enter passphrase"
                autoComplete="new-password"
              />
            </label>
            <p className="new-chat-dialog__hint">
              Beyond not being blank, there is no minimum length or complexity requirement -- it is
              your choice. A short or common passphrase is the only thing protecting this file if it
              leaks, so choose carefully.
            </p>

            {error ? (
              <p className="new-chat-dialog__error" role="alert">
                {error}
              </p>
            ) : null}

            <button type="submit" className="new-chat-dialog__submit" disabled={submitting}>
              {submitting ? "Encrypting..." : "Download backup"}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
