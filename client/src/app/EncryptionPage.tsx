import { Footer } from "@/components/layout/Footer";
import { SiteHeader } from "@/components/layout/SiteHeader";

const flow: ReadonlyArray<{ title: string; description: string }> = [
  {
    title: "Your browser",
    description: "Encrypts the message with the chat key. The text never leaves the device unencrypted.",
  },
  {
    title: "The server",
    description: "Stores and forwards the sealed envelope. It has no key to open it.",
  },
  {
    title: "Their browser",
    description: "Unwraps the chat key with the recipient's private key and decrypts the message.",
  },
];

const parameters: ReadonlyArray<[string, string]> = [
  ["Identity key pair", "Hybrid ML-KEM-768 + X25519 (post-quantum plus classical), generated in the browser"],
  ["Chat key", "256-bit random key per chat key version, from crypto.getRandomValues"],
  ["Chat key wrap", "Hybrid KEM encapsulation, then HKDF-SHA-256, then AES-256-GCM with a random 12-byte nonce"],
  ["Message encryption", "HKDF-SHA-256 from the chat key and key version id, then AES-256-GCM with a random 12-byte nonce"],
  ["Message size limit", "100 KiB of plaintext"],
  ["Identity at rest", "AES-256-GCM with a non-extractable Web Crypto key stored in IndexedDB"],
  ["Identity backup", "scrypt (N=2^17, r=8, p=1, random 16-byte salt) to a 256-bit key, then AES-256-GCM with a random 12-byte nonce"],
  ["Libraries", "Web Crypto for AES-GCM, @noble/post-quantum and @noble/hashes for ML-KEM, X25519, HKDF and scrypt"],
];

export function EncryptionPage() {
  return (
    <>
      <SiteHeader />
      <main className="legal encryption">
        <h1 className="legal__title">How encryption works</h1>
        <p className="legal__updated">What protects your messages, and what does not.</p>

        <h2>Overview</h2>
        <p>
          E2EE Text is end-to-end encrypted: messages are encrypted in your browser and decrypted
          only in the browsers of the people in the chat. The server stores public keys, wrapped
          chat keys and opaque ciphertext. It cannot read your messages.
        </p>

        <ol className="encryption__flow" aria-label="Message flow">
          {flow.map((step) => (
            <li key={step.title} className="encryption__step">
              <strong>{step.title}</strong>
              <span>{step.description}</span>
            </li>
          ))}
        </ol>

        <h2>Your identity keys</h2>
        <p>
          When you first sign in, your browser generates a key pair that combines ML-KEM, a
          post-quantum algorithm, with X25519, a classical one. An attacker would have to break
          both. The public key is uploaded so others can send you chat keys. The private key stays
          on your device, encrypted at rest with a key the browser will not let scripts export.
        </p>

        <h2>Chat keys</h2>
        <p>
          Every chat has a random symmetric key. It is sealed separately for each member using
          that member's public key, so only that member can open it. Each message is then
          encrypted with AES-256-GCM using a fresh random nonce, which also detects any tampering.
        </p>

        <h2>Key versions and rotation</h2>
        <p>
          If you lose your identity (cleared site data, a new browser, no backup restored), chat
          keys sealed for your old key cannot be opened. The next time you send in such a chat, a
          new chat key version is created and sealed for every member's current public key. Older
          messages on the new device show &ldquo;Message encrypted with a previous key&rdquo; until
          you restore your identity backup. Other members keep reading old messages as before.
        </p>

        <h2>Identity backup</h2>
        <p>
          You can export your identity to a file protected by a passphrase you choose. The
          passphrase is stretched with scrypt and the file is encrypted with AES-256-GCM. The
          passphrase never leaves your device, and we cannot recover it. Choose a strong one.
        </p>

        <h2>Limits and honest caveats</h2>
        <ul>
          <li>
            The server can see metadata: who talks to whom, when, and how large the messages are.
            Chat names are not encrypted.
          </li>
          <li>
            There is no forward secrecy ratchet. Anyone who obtains a chat key can read every
            message encrypted with that version.
          </li>
          <li>
            Removing a member does not rotate the chat key yet, so a removed member keeps what they
            already hold.
          </li>
          <li>
            Malicious script running in your open, unlocked tab could use your keys. Keep your
            browser and extensions trustworthy.
          </li>
          <li>
            You trust the JavaScript we serve. A compromised server could serve altered code.
          </li>
          <li>
            If you lose your identity and have no backup, old messages cannot be recovered.
          </li>
        </ul>

        <h2>Technical appendix</h2>
        <table className="encryption__table">
          <caption className="encryption__caption">Algorithms and parameters</caption>
          <thead>
            <tr>
              <th scope="col">Item</th>
              <th scope="col">Detail</th>
            </tr>
          </thead>
          <tbody>
            {parameters.map(([item, detail]) => (
              <tr key={item}>
                <th scope="row">{item}</th>
                <td>{detail}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </main>
      <Footer />
    </>
  );
}
