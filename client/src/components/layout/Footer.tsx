import { CONTACT_URL, donateUrl } from "@/lib/config";

export function Footer() {
  return (
    <footer className="site-foot">
      <p className="site-foot__support">
        E2EE Text will be sustained through{" "}
        <strong>advertisement</strong> and{" "}
        <a
          className="site-foot__donate"
          href={donateUrl}
          target="_blank"
          rel="noopener noreferrer"
        >
          <strong>donations</strong>
        </a>
      </p>
      <nav className="site-foot__nav" aria-label="Legal and contact">
        <a className="site-foot__link" href="/privacy">
          Privacy Policy
        </a>
        <a className="site-foot__link" href="/terms">
          Terms of Service
        </a>
        <a
          className="site-foot__link"
          href={CONTACT_URL}
          target="_blank"
          rel="noopener noreferrer"
        >
          Contact
        </a>
      </nav>
      <p className="site-foot__copy">&copy; 2026 e2eetext</p>
    </footer>
  );
}
