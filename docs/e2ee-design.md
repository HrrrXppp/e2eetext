# End-to-End Encryption Design (v1)

Status: implemented (v1). See the addendum at the bottom for implementation
notes that refine this design.

## Goals

- Server is a blind relay: it never holds plaintext or private key material.
- Resistant to both classical and quantum adversaries, including "harvest now,
  decrypt later" — traffic recorded today must stay unreadable even after a
  cryptographically relevant quantum computer exists.
- Scope: 1:1 chats and groups, single device per account.

## Cryptographic primitives

| Purpose | Algorithm | Notes |
|---|---|---|
| Key encapsulation (user keys, chat keys) | **ML-KEM-1024** (FIPS 203) | Pure post-quantum, no classical component (no X25519). Chosen over hybrid because with no classical fallback, the largest NIST security category (5) gives the most margin against future cryptanalysis of the lattice problem. |
| Message encryption | **AES-256-GCM** | Fresh random key per message. Symmetric crypto is already quantum-safe at 256-bit key size (Grover's algorithm only gives a quadratic speedup). |
| KDF | **HKDF-SHA-384** | Used wherever a shared secret needs to be turned into a wrapping/encryption key. |
| Signing / message authentication | **None in v1** | See Accepted risks. |

Client-side library: `@noble/post-quantum` (ML-KEM) + native WebCrypto (AES-GCM,
HKDF). No new server-side crypto dependency required for E2EE itself.

## Key model

- **User keypair**: ML-KEM-1024, generated client-side on account creation.
  Public key uploaded to the server; private key stays on the user's device.
- **Chat keypair**: ML-KEM-1024, generated client-side by whoever creates the
  chat (the admin). Public key stored server-side; private key is distributed
  to members via per-member key wrapping (see flow below) — the server only
  ever sees wrapped (encrypted) copies.
- Chat keys are **static for v1** — see Accepted risks.

## Data model changes

```sql
-- users: public KEM key
ALTER TABLE users ADD COLUMN kem_public_key BYTEA NOT NULL;

-- chats: creator + chat's public KEM key
ALTER TABLE chats ADD COLUMN admin_user_id UUID NOT NULL REFERENCES users (id);
ALTER TABLE chats ADD COLUMN kem_public_key BYTEA NOT NULL;

-- user_chats: each member's wrapped copy of the chat private key
ALTER TABLE user_chats ADD COLUMN wrapped_chat_private_key BYTEA;   -- AEAD ciphertext
ALTER TABLE user_chats ADD COLUMN kem_ciphertext BYTEA;              -- ML-KEM encapsulation output

-- messages: per-message hybrid encryption envelope
ALTER TABLE messages ADD COLUMN kem_ciphertext BYTEA NOT NULL;       -- wraps the message AES key
-- `data` continues to hold the AEAD ciphertext (nonce + tag + ciphertext)
```

`wrapped_chat_private_key` / `kem_ciphertext` on `user_chats` are nullable
until that member has received their copy.

## Flows

1. **User creation**: client generates an ML-KEM-1024 keypair. Private key
   stays on device; public key → `users.kem_public_key`.
2. **Chat creation**: creator's client generates the chat's ML-KEM-1024
   keypair. `chats.kem_public_key` = chat public key, creator becomes
   `admin_user_id`. Creator encapsulates against their own public key to wrap
   the chat private key for their own `user_chats` row
   (`kem_ciphertext` + AEAD-encrypted `wrapped_chat_private_key`).
3. **Adding a member**: admin's client fetches the new member's
   `kem_public_key`, encapsulates against it, AEAD-wraps the chat private key
   with the resulting shared secret, and writes `kem_ciphertext` +
   `wrapped_chat_private_key` into that member's `user_chats` row.
4. **Receiving access**: member's client reads their `user_chats` row,
   decapsulates `kem_ciphertext` with their own private key to recover the
   shared secret, AEAD-decrypts `wrapped_chat_private_key` to get the chat
   private key, and can now decrypt every message in that chat.
5. **Sending a message**: client generates a fresh random AES-256-GCM key,
   encrypts the message body with it, then encapsulates that key against
   `chats.kem_public_key`. `messages.kem_ciphertext` + `messages.data` store
   the two parts.

## Private key backup

Optional manual export. The user is not forced to back up their private key
during onboarding; they can export it (as a passphrase-protected encrypted
file) at any time from settings. If they never export it and lose their
device/storage, that account's chat history is permanently unrecoverable —
expected behavior for genuine E2EE, but must be stated clearly in the UI so
it isn't a surprise.

## Accepted risks (v1)

These are deliberate scope decisions for v1, not oversights. Both should be
tracked as follow-up work.

- **No message signing / authentication.** Messages are encrypted but not
  signed. A compromised or malicious server could forge messages, and could
  substitute a chat's public key during member-distribution to silently MITM
  a group. Mitigating this requires an identity signing keypair (e.g.
  ML-DSA-65) and per-message/per-distribution signatures — deferred.
- **No chat key rotation.** The chat keypair is generated once and never
  rotated. Removing a member from a chat does not revoke their ability to
  decrypt future messages, since they retain the chat private key they were
  given. Also means no forward secrecy or post-compromise security: a single
  compromise of the chat private key exposes all past and future messages
  until a rotation mechanism exists. Deferred; should be revisited before
  this is used for anything sensitive.

## Out of scope for this design

- Multi-device support (explicitly single device per account for now).
- Group membership fanout beyond the shared chat-key model (no per-sender
  keys, no ratcheting).
- Metadata protection (who talks to whom is visible to the server regardless
  of this design).
- Adding a member to an *existing* chat. Chat membership is fixed at creation
  time — every member's wrapped chat-key row is created atomically along with
  the chat itself. There is no endpoint to add a member later; a group's
  member list cannot change after creation in v1.

## Implementation addendum

The design above was refined in two ways while implementing it, without
changing any of the algorithms, accepted risks, or scope decisions:

**Crypto construction is HKDF-from-the-KEM-shared-secret, not a
separately-generated-then-wrapped key.** ML-KEM's `encapsulate(publicKey)`
returns its own random shared secret — it has no way to accept a caller-
supplied key to wrap. So "generate a fresh AES key, then encapsulate it"
(as flows 2 and 5 above describe) is implemented as: encapsulate first, then
`HKDF-SHA-384(sharedSecret, info=<domain string>)` to derive the AES-256-GCM
key directly. This is cryptographically equivalent — every `encapsulate` call
is independently random, so "a fresh key per message/wrap" still holds — it's
just precise about where the AES key actually comes from:

- Message key: `HKDF-SHA-384(sharedSecret, info="e2eetext/message-key/v1")`
- Chat-key wrap: `HKDF-SHA-384(sharedSecret, info="e2eetext/chat-key-wrap/v1")`

**`user_chats.wrapped_chat_private_key` / `kem_ciphertext` are `NOT NULL`,
not nullable.** Since adding a member to an existing chat is out of scope
(see above), every `user_chats` row is created atomically at chat-creation
time with its wrap data already populated — there's no "member exists but
hasn't received their key yet" intermediate state to model, so the columns
don't need to allow `NULL`.
