import { base64UrlToBytes, bytesToBase64Url } from "@/lib/e2ee/encoding";

// Wrapping keys are non-extractable CryptoKeys in IndexedDB. XSS cannot read
// raw key bytes from storage. Active XSS can still call WebCrypto/decrypt APIs
// while the tab is unlocked — full browser XSS isolation needs a passphrase
// lock or separate origin.
const DB_NAME = "e2eetext-local-crypto";
const DB_VERSION = 1;
const STORE_NAME = "wrappingKeys";

type WrappingKeyRecord = {
  userId: string;
  key: CryptoKey;
};

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "userId" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("failed to open local crypto db"));
  });
}

async function withStore<T>(
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest<T> | void,
): Promise<T> {
  const db = await openDatabase();
  try {
    return await new Promise<T>((resolve, reject) => {
      const tx = db.transaction(STORE_NAME, mode);
      const store = tx.objectStore(STORE_NAME);
      const request = run(store);
      tx.onerror = () => reject(tx.error ?? new Error("local crypto transaction failed"));
      if (request) {
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error ?? new Error("local crypto request failed"));
      } else {
        tx.oncomplete = () => resolve(undefined as T);
      }
    });
  } finally {
    db.close();
  }
}

async function generateWrappingKey(): Promise<CryptoKey> {
  return crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, ["encrypt", "decrypt"]);
}

async function loadWrappingKey(userId: string): Promise<CryptoKey | null> {
  const record = await withStore<WrappingKeyRecord | undefined>("readonly", (store) => store.get(userId));
  return record?.key ?? null;
}

async function saveWrappingKey(userId: string, key: CryptoKey): Promise<void> {
  const record: WrappingKeyRecord = { userId, key };
  await withStore("readwrite", (store) => store.put(record));
}

async function getStorageKey(userId: string): Promise<CryptoKey> {
  const existing = await loadWrappingKey(userId);
  if (existing) {
    return existing;
  }

  const key = await generateWrappingKey();
  await saveWrappingKey(userId, key);
  return key;
}

export async function encryptLocalValue(userId: string, value: string): Promise<string> {
  const key = await getStorageKey(userId);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: iv.buffer as ArrayBuffer },
    key,
    new TextEncoder().encode(value).buffer as ArrayBuffer,
  );

  return JSON.stringify({
    v: 1,
    iv: bytesToBase64Url(iv),
    ciphertext: bytesToBase64Url(new Uint8Array(ciphertext)),
  });
}

export async function decryptLocalValue(userId: string, payload: string): Promise<string> {
  const parsed = JSON.parse(payload) as { v?: number; iv?: string; ciphertext?: string };
  if (parsed.v !== 1 || !parsed.iv || !parsed.ciphertext) {
    throw new Error("invalid encrypted local value");
  }

  const key = await getStorageKey(userId);
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: base64UrlToBytes(parsed.iv).buffer as ArrayBuffer },
    key,
    base64UrlToBytes(parsed.ciphertext).buffer as ArrayBuffer,
  );

  return new TextDecoder().decode(plaintext);
}

export async function clearLocalCryptoKey(userId: string): Promise<void> {
  await withStore("readwrite", (store) => store.delete(userId));
}
