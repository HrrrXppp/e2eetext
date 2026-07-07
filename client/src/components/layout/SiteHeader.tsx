import { useState } from "react";
import { EditNameDialog } from "@/components/auth/EditNameDialog";
import { KeyBackupDialog } from "@/components/auth/KeyBackupDialog";
import { SignInDialog } from "@/components/auth/SignInDialog";
import { useAuth } from "@/hooks/useAuth";
import { appVersion } from "@/lib/version";
import { userDisplayName } from "@/lib/users";

export function SiteHeader() {
  const { user, providers, loading, signOut, setDisplayName } = useAuth();
  const [signInOpen, setSignInOpen] = useState(false);
  const [editNameOpen, setEditNameOpen] = useState(false);
  const [keyBackupOpen, setKeyBackupOpen] = useState(false);
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
              <a
                className={`site-head__sign-in site-head__chats${onChatsPage ? " site-head__chats--active" : ""}`}
                href="/chats"
                aria-current={onChatsPage ? "page" : undefined}
              >
                Chats
              </a>
              <button
                type="button"
                className="site-head__sign-in"
                onClick={() => setKeyBackupOpen(true)}
              >
                Backup keys
              </button>
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

      {keyBackupOpen && user ? (
        <KeyBackupDialog userId={user.id} onClose={() => setKeyBackupOpen(false)} />
      ) : null}
    </>
  );
}
