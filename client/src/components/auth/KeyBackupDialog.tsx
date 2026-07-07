import { FormEvent, useEffect, useId, useRef, useState } from "react";
import { exportOwnKeyPairEncrypted, importOwnKeyPairFromBackup } from "@/lib/keyBackup";
import { saveOwnKeyPair } from "@/lib/keyStore";

type KeyBackupDialogProps = {
  userId: string;
  onClose: () => void;
  onRestored?: () => void;
};

export function KeyBackupDialog({ userId, onClose, onRestored }: KeyBackupDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [exportPassphrase, setExportPassphrase] = useState("");
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<string | null>(null);
  const [exportDone, setExportDone] = useState(false);

  const [restoreFile, setRestoreFile] = useState<File | null>(null);
  const [restorePassphrase, setRestorePassphrase] = useState("");
  const [restoring, setRestoring] = useState(false);
  const [restoreError, setRestoreError] = useState<string | null>(null);

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

  async function handleExport(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setExporting(true);
    setExportError(null);
    setExportDone(false);

    try {
      const blob = await exportOwnKeyPairEncrypted(userId, exportPassphrase);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${userId}-e2ee-backup.json`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
      setExportDone(true);
    } catch {
      setExportError("Could not export your key. Try again.");
    } finally {
      setExporting(false);
    }
  }

  async function handleRestore(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!restoreFile) {
      setRestoreError("Choose a backup file first.");
      return;
    }

    setRestoring(true);
    setRestoreError(null);

    try {
      const keyPair = await importOwnKeyPairFromBackup(restoreFile, restorePassphrase);
      await saveOwnKeyPair(userId, keyPair);
      onRestored?.();
      onClose();
    } catch {
      setRestoreError("Could not restore this backup. Check the file and passphrase.");
    } finally {
      setRestoring(false);
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
          aria-label="Close key backup dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Encryption key backup
        </h2>
        <p className="sign-in-dialog__lead">
          Your private key never leaves this device unless you export it. Losing it means losing
          access to your encrypted chat history &mdash; there is no server-side recovery.
        </p>

        <form className="new-chat-dialog__form" onSubmit={handleExport}>
          <h3 className="new-chat-dialog__members-label">Download backup</h3>
          <label className="new-chat-dialog__field">
            <span>Passphrase</span>
            <input
              type="password"
              value={exportPassphrase}
              onChange={(event) => setExportPassphrase(event.target.value)}
              placeholder="Choose a strong passphrase"
              autoComplete="new-password"
            />
          </label>

          {exportError ? (
            <p className="new-chat-dialog__error" role="alert">
              {exportError}
            </p>
          ) : null}
          {exportDone ? <p className="new-chat-dialog__hint">Backup downloaded.</p> : null}

          <button
            type="submit"
            className="new-chat-dialog__submit"
            disabled={exporting || !exportPassphrase}
          >
            {exporting ? "Exporting..." : "Download backup"}
          </button>
        </form>

        <form className="new-chat-dialog__form" onSubmit={handleRestore}>
          <h3 className="new-chat-dialog__members-label">Restore from backup</h3>
          <label className="new-chat-dialog__field">
            <span>Backup file</span>
            <input
              type="file"
              accept="application/json"
              onChange={(event) => setRestoreFile(event.target.files?.[0] ?? null)}
            />
          </label>
          <label className="new-chat-dialog__field">
            <span>Passphrase</span>
            <input
              type="password"
              value={restorePassphrase}
              onChange={(event) => setRestorePassphrase(event.target.value)}
              autoComplete="current-password"
            />
          </label>

          {restoreError ? (
            <p className="new-chat-dialog__error" role="alert">
              {restoreError}
            </p>
          ) : null}

          <button
            type="submit"
            className="new-chat-dialog__submit"
            disabled={restoring || !restoreFile || !restorePassphrase}
          >
            {restoring ? "Restoring..." : "Restore from backup"}
          </button>
        </form>
      </div>
    </div>
  );
}
