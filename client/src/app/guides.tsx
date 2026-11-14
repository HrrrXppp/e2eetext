import type { ReactNode } from "react";

export type Guide = {
  path: string;
  title: string;
  navLabel: string;
  summary: string;
  body: ReactNode;
};

export const guides: readonly Guide[] = [
  {
    path: "/guides/what-the-server-can-see",
    title: "What the server can and cannot read",
    navLabel: "What the server can see",
    summary: "Plaintext stays in the browsers. The server still stores accounts, names, and ciphertext.",
    body: (
      <>
        <p>
          A message is encrypted in your browser before it is sent, and it is decrypted only in the
          browsers of the people in that chat. The server stores the sealed result. It does not have
          the chat key, and it does not have your private identity key, so it cannot turn that sealed
          result back into text.
        </p>
        <h2>What never reaches the server</h2>
        <p>
          The text of a message is encrypted with AES-256-GCM under a key that belongs to the chat.
          What is stored is an envelope: a version, the algorithm name, the id of the chat-key version,
          a random nonce, and the ciphertext. That envelope is opaque. The backup passphrase you choose
          for your identity file is stretched and used on your device. It is not uploaded, and we cannot
          reset it.
        </p>
        <p>
          Your private identity key stays in the browser. The copy kept for the next visit is itself
          encrypted with a key the browser will not let a script export. Sign-out removes that account&apos;s
          local encryption material from the browser.
        </p>
        <h2>What the server does store</h2>
        <p>
          The server has to deliver messages, so it keeps the information that delivery needs. When you
          sign in with an OpenID Connect provider, such as Google, we receive that provider&apos;s account
          identifier and, when the provider sends one, a display name. You can change the display name
          later. Your public identity key is stored so other people can seal chat keys to you.
        </p>
        <p>
          For each chat the server stores the chat name in the clear, the list of members, and each
          member&apos;s wrapped copy of the chat key. The wrap is sealed to that member&apos;s public key.
          The server cannot open it. For each message it stores who sent it, when it was created, and the
          ciphertext, which also reveals the size of the message. Desktop notifications, when you allow
          them, use the chat name only. The notification text is &ldquo;New message in&rdquo; that name.
          The message body is not included.
        </p>
        <h2>What that means in practice</h2>
        <p>
          Someone who takes a copy of the database can see who has an account, which display name they
          use, who shares a chat, when messages were sent, how large those messages were, and the chat
          names. They cannot read the messages. The same limit applies to us. Metadata is not hidden, and
          the encryption page states that directly.
        </p>
      </>
    ),
  },
  {
    path: "/guides/chat-keys",
    title: "How a chat key and a new key version work",
    navLabel: "Chat keys",
    summary: "Each chat version has its own key, sealed separately for every member.",
    body: (
      <>
        <p>
          A chat does not reuse your identity key to encrypt messages. Identity keys only wrap a
          separate chat key. The chat key is a fresh 256-bit random value from the browser&apos;s
          cryptographic random source. Each message is then encrypted with AES-256-GCM under a key
          derived from that chat key and the id of its version, with a new random 12-byte nonce every
          time. The nonce and the authentication tag mean a modified ciphertext does not decrypt.
        </p>
        <h2>How a member receives the chat key</h2>
        <p>
          When a chat is created, the browser of the person creating it generates the chat key and a
          key id. It fetches each member&apos;s current public identity key and seals a separate copy
          of the chat key to each of them. The seal is a hybrid of ML-KEM-768 and X25519, then
          HKDF-SHA-256, then AES-256-GCM. An attacker has to break both the post-quantum algorithm and
          the classical one. The server stores those sealed copies. Only the member whose private key
          matches a copy can open it.
        </p>
        <p>
          Opening a chat downloads the sealed copies and unwraps the one addressed to you. That unwrapped
          key is cached on the device, wrapped again with the non-extractable browser key, so later
          messages in the same version do not need another trip through the identity key.
        </p>
        <h2>When a new version appears</h2>
        <p>
          A chat key version is the pair of a key id and the chat key sealed for the members at that
          moment. Messages name the version they used. A message always decrypts with that version, not
          with whatever the newest version happens to be.
        </p>
        <p>
          A new version is created when the identity on a device can no longer open the current one.
          That happens if site data was cleared, or you are in a new browser, and you did not restore
          your identity backup. The next time you send in that chat, your browser makes a new chat key,
          seals it for every current member&apos;s current public key, and uploads it. The message you
          are sending uses the new version. Older versions stay. Other members keep reading old messages
          with the key they already hold, and they pick up the new version when a message arrives with a
          key id they have not seen. On the device that lost its identity, older messages show
          &ldquo;Message encrypted with a previous key&rdquo; until that identity is restored.
        </p>
        <h2>What a version does not do</h2>
        <p>
          There is no forward-secrecy ratchet. Anyone who obtains a chat key can read every message
          encrypted with that version, including messages sent earlier under the same version. Removing
          a member does not create a new version yet, so a removed member keeps any chat key they already
          unwrapped and can still read messages they already fetched. Any current member can add a
          version. If two members do that at the same time, both versions remain readable, and the newest
          by creation time is the one used for the next send.
        </p>
      </>
    ),
  },
  {
    path: "/guides/disappearing-messages",
    title: "How disappearing messages are timed and deleted",
    navLabel: "Disappearing messages",
    summary: "One duration for the chat, counted from the moment each message was sent.",
    body: (
      <>
        <p>
          Disappearing messages are a rule of the chat, not a timer stored on each message. A new chat
          keeps messages for 60 days. The create screen tells you the date and time a message sent at
          that moment would disappear. Sixty days is 86,400 minutes, and that is the duration stored
          for the chat.
        </p>
        <h2>When a message expires</h2>
        <p>
          A message expires at the time it was created plus the chat&apos;s duration. A message sent
          today and a message sent next week do not disappear together. Each one lasts 60 days from its
          own send time. The app shows that time on the message and drops the message from the open chat
          once the time has passed, without waiting for the next visit.
        </p>
        <p>
          The server enforces the same rule by deleting the row. A scheduled job runs about once a
          minute and hard-deletes every message whose send time plus the chat duration is in the past.
          There is no expiry column on the message and no leftover plaintext. The server never had the
          text. After those messages are gone, a chat-key version that no remaining message uses is
          deleted as well, together with the sealed copies of that version. The newest version of the
          chat is kept, so members can still send into a chat whose messages have all expired.
        </p>
        <h2>One duration for every message already sent</h2>
        <p>
          Because the duration lives on the chat, it applies to every message in that chat. The product
          sets 60 days when the chat is created. The duration is not chosen message by message, and the
          create screen does not offer a different length. If that chat-level duration were changed,
          messages already sent would be judged against the new duration, because their expiry is
          calculated from the send time and the chat&apos;s current duration rather than from a date
          frozen on the message.
        </p>
      </>
    ),
  },
  {
    path: "/guides/identity-backup",
    title: "What identity backup does and does not protect",
    navLabel: "Identity backup",
    summary: "A passphrase-encrypted file is the only way to restore your key. We cannot recover it.",
    body: (
      <>
        <p>
          Your identity is a key pair generated in the browser the first time you sign in on a device.
          The public half is uploaded so other people can seal chat keys to you. The private half stays
          on that device. If you clear site data, use another browser, or lose the device, that private
          key is gone unless you have a backup. Chat keys sealed to the lost key cannot be opened, and
          old messages on the new device stay unreadable.
        </p>
        <h2>How to make a backup</h2>
        <p>
          Open your name in the header. <strong>Back up key</strong> asks for a passphrase that is
          separate from your sign-in. There is no minimum length. The dialog shows Weak, Fair, or Strong
          as you type, and it does not block a weak passphrase. Choose one you can keep. The passphrase
          is stretched with scrypt (N = 2^17, r = 8, p = 1, and a random 16-byte salt) into a 256-bit
          key, and the identity is encrypted with AES-256-GCM under a random nonce. The file downloads
          to your computer. It is not copied to the clipboard, and it is not sent to the server.
        </p>
        <p>
          The same dialog is offered once, immediately after a new identity is generated: a new account,
          or the first sign-in on a browser that does not already have a key. It is the moment the key
          exists only in that browser. The lead text on the dialog states the limit in plain words: the
          key lives only on this device, the server can never recover it, and the downloaded file is the
          only way to restore it.
        </p>
        <h2>How restore works</h2>
        <p>
          <strong>Restore key</strong> takes the file and the passphrase. A wrong passphrase and a
          damaged file fail the same way, so the error is not a hint about the passphrase. A backup
          made from a different account is rejected. Backups are bound to the account they were exported
          from, and that binding sits inside the encrypted part of the file.
        </p>
        <p>
          Restore replaces the private key on this device. If this device or the account&apos;s current
          public key already differs from the file,           the dialog stops and asks you to confirm <strong>Overwrite and restore</strong>. That confirmation tells you that other people&apos;s
          existing chat-key wraps are bound to a specific public key and may have to be re-shared.
          Restoring after you lose a device recovers your identity. It is not a way to keep two devices
          in sync. The account is set up for one device.
        </p>
        <h2>What the backup does not cover</h2>
        <p>
          We never see the passphrase, so we cannot reset it and we cannot read the file. Anyone who has
          both the file and the passphrase can restore the identity and read messages that identity can
          open. Store them apart from each other. A backup does not hide metadata from the server, and it
          does not recover a chat key that was never sealed to this identity. If you lose both the device
          and the backup, messages sealed to that key cannot be recovered.
        </p>
      </>
    ),
  },
  {
    path: "/guides/who-you-are-talking-to",
    title: "How to confirm who you are talking to",
    navLabel: "Who you are talking to",
    summary: "A display name is not a verified identity. The app does not show a key fingerprint.",
    body: (
      <>
        <p>
          You add someone to a chat by searching the display name they chose. Search starts at three
          characters. The name is theirs to set and theirs to change. It is not checked against a
          government id, and it is not a cryptographic identity. Two people can use similar names. The
          sign-in provider, for example Google, attests that a particular provider account signed in. It
          does not attest that the person at the keyboard is the person you meant, and it does not freeze
          the encryption key to that person.
        </p>
        <h2>What the app checks instead</h2>
        <p>
          When a chat key is sealed, the browser fetches each member&apos;s current public key and seals
          to that key. Delivery of later messages still requires the private key that matches. That
          protects the contents from the server and from anyone who does not hold a member&apos;s private
          key. It does not tell you whether the public key belongs to the person you think it does.
        </p>
        <p>
          The app has no safety number, no QR comparison, and no screen that shows a fingerprint of
          someone&apos;s public key. There is also no warning when someone&apos;s public key changes.
          A person who clears site data, or who signs in on a new browser without restoring a backup,
          publishes a new public key. The next message they send in an existing chat creates a new chat-key
          version sealed to the keys on file at that moment, including the new one. Other members are not
          asked to compare the old key with the new one.
        </p>
        <h2>How to be sure anyway</h2>
        <p>
          Confirm the person outside the app. Agree on the display name in a channel you already trust,
          and treat a changed name or a sudden new device as a reason to check again with them directly.
          Removing someone from a chat does not rotate the chat key yet, so removal is not proof they can
          no longer read messages they already received. Until the app can show you a key fingerprint to
          compare, the name in the member list is a label, not a verification.
        </p>
      </>
    ),
  },
];

export function guideByPath(pathname: string): Guide | undefined {
  return guides.find((guide) => guide.path === pathname);
}
