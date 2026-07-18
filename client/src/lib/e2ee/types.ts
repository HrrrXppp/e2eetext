export const E2EE_IDENTITY_ALG = "hybrid-kem-mlkem768-x25519";
export const E2EE_WRAP_ALG = "hybrid-kem-mlkem768-x25519-aes256gcm";
export const E2EE_MESSAGE_ALG = "aes256gcm-chat-key";
export const MAX_MESSAGE_PLAINTEXT_BYTES = 102_400;

export type IdentityPublicKey = {
  v: 1;
  alg: typeof E2EE_IDENTITY_ALG;
  publicKey: string;
};

export type StoredIdentity = {
  publicKey: string;
  secretKey: string;
};

export type ChatKeyWrap = {
  v: 1;
  alg: typeof E2EE_WRAP_ALG;
  keyId: string;
  kemCiphertext: string;
  nonce: string;
  ciphertext: string;
};

export type MessageEnvelope = {
  v: 1;
  alg: typeof E2EE_MESSAGE_ALG;
  keyId: string;
  nonce: string;
  ciphertext: string;
};

export type ChatKeyWrapResponse = {
  keyId: string;
  wrap: ChatKeyWrap;
  createdAt: string;
};
