import { useState } from "react";
import { BackupIdentityDialog } from "@/components/auth/BackupIdentityDialog";
import { EditNameDialog } from "@/components/auth/EditNameDialog";
import { IdentityBackupPrompt } from "@/components/auth/IdentityBackupPrompt";
import { RestoreIdentityDialog } from "@/components/auth/RestoreIdentityDialog";
import { SignInDialog } from "@/components/auth/SignInDialog";
import { useAuth } from "@/hooks/useAuth";
import { appVersion } from "@/lib/version";
import { userDisplayName } from "@/lib/users";

export function SiteHeader() {
  const {
    user,
    providers,
    loading,
    signOut,
    setDisplayName,
    justCreatedIdentity,
    acknowledgeIdentityBackup,
  } = useAuth();
  const [signInOpen, setSignInOpen] = useState(false);
  const [editNameOpen, setEditNameOpen] = useState(false);
  const [backupOpen, setBackupOpen] = useState(false);
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [skipProfile, setSkipProfile] = useState(false);
  const onChatsPage =
    window.location.pathname === "/chats" || window.location.pathname.startsWith("/chats/");

  return (
    <>
      <header className="site-head">
        <div className="site-head__inner">
          <a className="site-head__brand-group" href="/">
            <span className="site-head__mark" aria-hidden="true">
              <svg viewBox="0 0 32 36" fill="none">
                <rect x="8" y="16" width="16" height="14" rx="3" fill="currentColor" />
                <path
                  d="M11 16V12a5 5 0 0 1 10 0v4"
                  stroke="currentColor"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                />
              </svg>
            </span>
            <p className="site-head__brand">E2EE Text</p>
            <span className="site-head__version" title="Application version">
              version: {appVersion}
            </span>
          </a>
          {loading ? (
            <span className="site-head__sign-in site-head__sign-in--loading">Loading...</span>
          ) : user ? (
            <div className="site-head__user">
              <div className="site-head__user-identity">
                <span className="site-head__user-caption">Name</span>
                <button
                  type="button"
                  className={`site-head__user-name${
                    user.name?.trim() ? "" : " site-head__user-name--placeholder"
                  }`}
                  onClick={() => setEditNameOpen(true)}
                  title="Edit display name"
                >
                  {userDisplayName(user.name)}
                </button>
              </div>
              <button
                type="button"
                className="site-head__sign-in"
                onClick={() => setBackupOpen(true)}
                title="Download an encrypted backup of your private key"
              >
                Back up key
              </button>
              <button
                type="button"
                className="site-head__sign-in"
                onClick={() => setRestoreOpen(true)}
                title="Restore your private key from a backup file"
              >
                Restore key
              </button>
              <a
                className={`site-head__sign-in site-head__chats${onChatsPage ? " site-head__chats--active" : ""}`}
                href="/chats"
                aria-current={onChatsPage ? "page" : undefined}
              >
                Chats
              </a>
              <button type="button" className="site-head__sign-in" onClick={signOut}>
                Sign out
              </button>
            </div>
          ) : providers.length > 0 ? (
            <button
              type="button"
              className="site-head__sign-in"
              onClick={() => setSignInOpen(true)}
            >
              Sign in
            </button>
          ) : (
            <span className="site-head__sign-in site-head__sign-in--disabled" title="No sign-in providers">
              Sign in
            </span>
          )}
        </div>
      </header>

      {signInOpen ? (
        <SignInDialog
          providers={providers}
          skipProfile={skipProfile}
          onSkipProfileChange={setSkipProfile}
          onClose={() => setSignInOpen(false)}
        />
      ) : null}

      {editNameOpen && user ? (
        <EditNameDialog
          userId={user.id}
          currentName={user.name}
          onClose={() => setEditNameOpen(false)}
          onSaved={(name) => {
            setDisplayName(name);
            setEditNameOpen(false);
          }}
        />
      ) : null}

      {backupOpen && user ? (
        <BackupIdentityDialog userId={user.id} onClose={() => setBackupOpen(false)} />
      ) : null}

      {restoreOpen && user ? (
        <RestoreIdentityDialog userId={user.id} onClose={() => setRestoreOpen(false)} />
      ) : null}

      {justCreatedIdentity && user && !backupOpen && !restoreOpen ? (
        <IdentityBackupPrompt userId={user.id} onDismiss={acknowledgeIdentityBackup} />
      ) : null}
    </>
  );
}
