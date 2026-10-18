import { LegalPage } from "@/app/LegalPage";
import { CONTACT_URL } from "@/lib/config";

export function PrivacyPolicyPage() {
  return (
    <LegalPage title="Privacy Policy" updated="October 17, 2026">
      <p>
        E2EE Text is an end-to-end encrypted, web-based messenger. This page describes what information the service handles.
      </p>

      <h2>Information we collect</h2>
      <ul>
        <li>
          <strong>Sign-in identity.</strong> When you sign in with an OpenID Connect
          provider (for example Google) we receive your provider account identifier and
          optional display name.
        </li>
        <li>
          <strong>Encrypted message data.</strong> Messages are encrypted in your browser.
          Our servers store only wrapped chat keys and opaque ciphertext, never plaintext,
          and cannot read your conversations. Messages are disappeared by design.
        </li>
        <li>
          <strong>Public encryption key.</strong> Your public key is stored so other
          participants can encrypt chats to you. Your private key stays in your browser.
        </li>
      </ul>

      <h2>Information stored in your browser</h2>
      <p>
        The app uses your browser&apos;s <code>localStorage</code> to keep your sign-in
        token and your encryption identity. The app does not currently set cookies.
      </p>

      <h2>Advertising and cookies</h2>
      <p>
        E2EE Text is intended to be supported by advertising and donations. When ads are
        enabled, third-party vendors, including Google, may use cookies to serve ads based
        on your prior visits to this and other websites. You can opt out of personalized
        advertising at{" "}
        <a href="https://www.google.com/settings/ads" target="_blank" rel="noopener noreferrer">
          Google Ads Settings
        </a>
        . Ads are never shown inside your encrypted conversations.
      </p>

      <h2>How information is used</h2>
      <p>
        Information is used only to sign you in, deliver your messages and operate the
        service. We do not sell personal information.
      </p>

      <h2>Your choices</h2>
      <p>
        You can clear locally stored data through your browser settings. To request
        deletion of your account data, contact us using the link below.
      </p>

      <h2>Contact</h2>
      <p>
        Questions about this policy:{" "}
        <a href={CONTACT_URL} target="_blank" rel="noopener noreferrer">
          open an issue on GitHub
        </a>
        .
      </p>
    </LegalPage>
  );
}
