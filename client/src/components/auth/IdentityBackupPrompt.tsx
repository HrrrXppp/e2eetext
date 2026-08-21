import { useId, useState } from "react";
import { BackupIdentityDialog } from "@/components/auth/BackupIdentityDialog";
import { RestoreIdentityDialog } from "@/components/auth/RestoreIdentityDialog";

type IdentityBackupPromptProps = {
  userId: string;
  onDismiss: () => void;
};

type PromptMode = "prompt" | "backup" | "restore";

// One-time prompt shown right after a brand-new local identity keypair is
// generated (new account, or a new device with no local identity yet --
// see identityJustGenerated in useAuth.tsx). It is not gated on a
// persisted "seen" flag: it fires once per generation event and never
// reappears on later sign-ins, by design (issue #52 review feedback).
export function IdentityBackupPrompt({ userId, onDismiss }: IdentityBackupPromptProps) {
  const titleId = useId();
  const [mode, setMode] = useState<PromptMode>("prompt");

  if (mode === "backup") {
    return <BackupIdentityDialog userId={userId} onClose={onDismiss} />;
  }

  if (mode === "restore") {
    return <RestoreIdentityDialog userId={userId} onClose={onDismiss} />;
  }

  return (
    <div className="sign-in-dialog__backdrop" onClick={onDismiss}>
      <div
        className="sign-in-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(event) => event.stopPropagation()}
      >
        <button
          type="button"
          className="sign-in-dialog__close"
          aria-label="Skip backup for now"
          onClick={onDismiss}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Save a backup of your private key
        </h2>
        <p className="sign-in-dialog__lead">
          A new private key was just created for this account. It lives only on this device, and the
          server can never recover it -- if this device is lost or its storage is cleared, your message
          history becomes unreadable unless you have a backup. Restoring from an existing backup instead
          is also available below.
        </p>

        <button type="button" className="new-chat-dialog__submit" onClick={() => setMode("backup")}>
          Back up now
        </button>
        <button type="button" className="new-chat-dialog__back" onClick={() => setMode("restore")}>
          Restore from a backup instead
        </button>
        <button type="button" className="new-chat-dialog__back" onClick={onDismiss}>
          Skip for now
        </button>
      </div>
    </div>
  );
}
