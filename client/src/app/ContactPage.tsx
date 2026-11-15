import { useState } from "react";
import { Footer } from "@/components/layout/Footer";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { CONTACT_URL, HELP_EMAIL, MARKETING_EMAIL } from "@/lib/config";

function EmailLink({ address }: { address: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }

  return (
    <p>
      <a href={`mailto:${address}`}>{address}</a>{" "}
      <button type="button" className="contact__copy" onClick={copy} aria-label={`Copy ${address}`}>
        {copied ? "Copied" : "Copy"}
      </button>
    </p>
  );
}

export function ContactPage() {
  return (
    <>
      <SiteHeader />
      <main className="legal contact">
        <h1 className="legal__title">Contact</h1>
        <p className="legal__updated">How to reach us.</p>

        <h2>Help and support</h2>
        <p>
          Write to us about account trouble, bugs, or questions about how the encryption works.
        </p>
        <EmailLink address={HELP_EMAIL} />

        <h2>Marketing and cooperation</h2>
        <p>Write to us about partnerships, press, and advertising.</p>
        <EmailLink address={MARKETING_EMAIL} />

        <h2>Bugs and feature requests</h2>
        <p>
          Public issues are the best place for bugs and ideas:{" "}
          <a href={CONTACT_URL} target="_blank" rel="noopener noreferrer">
            GitHub issues
          </a>
          . Never post anything private or sensitive there, such as keys or message content.
        </p>

        <h2>Response time</h2>
        <p>We read every message and reply as soon as we can.</p>
      </main>
      <Footer />
    </>
  );
}
