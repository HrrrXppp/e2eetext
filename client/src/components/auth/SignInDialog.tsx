import { useEffect, useId, useRef } from "react";
import type { OIDCProvider } from "@/hooks/useAuth";
import { API_V1 } from "@/lib/api";
import { setSkipProfileOnCreate } from "@/lib/signInPreferences";

type SignInDialogProps = {
  providers: OIDCProvider[];
  skipProfile: boolean;
  onSkipProfileChange: (skip: boolean) => void;
  onClose: () => void;
};

export function SignInDialog({
  providers,
  skipProfile,
  onSkipProfileChange,
  onClose,
}: SignInDialogProps) {
  const titleId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);

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

  function handleProviderLogin(slug: string) {
    setSkipProfileOnCreate(skipProfile);
    window.location.href = `${API_V1}/auth/${slug}/login`;
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
          aria-label="Close sign in dialog"
          onClick={onClose}
        >
          ×
        </button>

        <h2 id={titleId} className="sign-in-dialog__title">
          Sign in
        </h2>
        <p className="sign-in-dialog__lead">Choose a provider to continue.</p>

        <ul className="sign-in-dialog__providers">
          {providers.map((provider) => (
            <li key={provider.id}>
              <button
                type="button"
                className="sign-in-dialog__provider"
                onClick={() => handleProviderLogin(provider.slug)}
              >
                Sign in with {provider.name}
              </button>
            </li>
          ))}
        </ul>

        <label className="sign-in-dialog__checkbox">
          <input
            type="checkbox"
            checked={skipProfile}
            onChange={(event) => onSkipProfileChange(event.target.checked)}
          />
          <span>Don&apos;t save name if user will be created</span>
        </label>
      </div>
    </div>
  );
}
