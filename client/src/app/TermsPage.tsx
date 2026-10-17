import { LegalPage } from "@/app/LegalPage";
import { CONTACT_URL } from "@/components/layout/Footer";

export function TermsPage() {
  return (
    <LegalPage title="Terms of Service" updated="October 17, 2026">
      <p>By using E2EE Text you agree to these terms.</p>

      <h2>The service</h2>
      <p>
        E2EE Text is a web-based messenger with end-to-end encryption. You sign in through
        a supported OpenID Connect provider and are responsible for keeping your account
        and your exported encryption key safe. Because messages are end-to-end encrypted,
        we cannot recover them if you lose your key.
      </p>

      <h2>Acceptable use</h2>
      <p>
        You agree not to use the service for anything unlawful, to harass or harm others,
        to distribute malware or spam, or to attempt to disrupt or gain unauthorized access
        to the service.
      </p>

      <h2>Disclaimer of warranty</h2>
      <p>
        The service is provided &quot;AS IS&quot;, without warranty of any kind, express or
        implied, including merchantability, fitness for a particular purpose and
        non-infringement.
      </p>

      <h2>Limitation of liability</h2>
      <p>
        To the maximum extent permitted by law, the operator is not liable for any claim,
        damages or other liability arising from your use of the service.
      </p>

      <h2>Abuse reports and contact</h2>
      <p>
        To report abuse or ask a question,{" "}
        <a href={CONTACT_URL} target="_blank" rel="noopener noreferrer">
          open an issue on GitHub
        </a>
        .
      </p>
    </LegalPage>
  );
}
