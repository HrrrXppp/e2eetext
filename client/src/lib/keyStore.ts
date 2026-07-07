import type { KemKeyPair } from "./crypto";

const DB_NAME = "messenger-e2ee-keys";
const DB_VERSION = 1;
const OWN_KEY_PAIR_STORE = "ownKeyPair";
const CHAT_KEYS_STORE = "chatKeys";

type OwnKeyPairRecord = {
  userId: string;
  publicKey: Uint8Array;
  secretKey: Uint8Array;
  createdAt: number;
};

type ChatKeyRecord = {
  id: string;
  userId: string;
  chatId: string;
  chatPrivateKey: Uint8Array;
  updatedAt: number;
};

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);

    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(OWN_KEY_PAIR_STORE)) {
        db.createObjectStore(OWN_KEY_PAIR_STORE, { keyPath: "userId" });
      }
      if (!db.objectStoreNames.contains(CHAT_KEYS_STORE)) {
        db.createObjectStore(CHAT_KEYS_STORE, { keyPath: "id" });
      }
    };

    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

async function withStore<T>(
  storeName: string,
  mode: IDBTransactionMode,
  run: (store: IDBObjectStore) => IDBRequest<T> | void,
): Promise<T> {
  const db = await openDatabase();
  try {
    return await new Promise<T>((resolve, reject) => {
      const tx = db.transaction(storeName, mode);
      const store = tx.objectStore(storeName);
      const request = run(store);

      tx.onerror = () => reject(tx.error);
      if (request) {
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
      } else {
        tx.oncomplete = () => resolve(undefined as T);
      }
    });
  } finally {
    db.close();
  }
}

export async function saveOwnKeyPair(userId: string, keyPair: KemKeyPair): Promise<void> {
  const record: OwnKeyPairRecord = {
    userId,
    publicKey: keyPair.publicKey,
    secretKey: keyPair.secretKey,
    createdAt: Date.now(),
  };
  await withStore(OWN_KEY_PAIR_STORE, "readwrite", (store) => store.put(record));
}

export async function loadOwnKeyPair(userId: string): Promise<KemKeyPair | null> {
  const record = await withStore<OwnKeyPairRecord | undefined>(OWN_KEY_PAIR_STORE, "readonly", (store) =>
    store.get(userId),
  );
  if (!record) {
    return null;
  }
  return { publicKey: record.publicKey, secretKey: record.secretKey };
}

export async function saveChatPrivateKey(userId: string, chatId: string, chatPrivateKey: Uint8Array): Promise<void> {
  const record: ChatKeyRecord = {
    id: `${userId}:${chatId}`,
    userId,
    chatId,
    chatPrivateKey,
    updatedAt: Date.now(),
  };
  await withStore(CHAT_KEYS_STORE, "readwrite", (store) => store.put(record));
}

export async function loadChatPrivateKey(userId: string, chatId: string): Promise<Uint8Array | null> {
  const record = await withStore<ChatKeyRecord | undefined>(CHAT_KEYS_STORE, "readonly", (store) =>
    store.get(`${userId}:${chatId}`),
  );
  return record ? record.chatPrivateKey : null;
}

export async function clearAllKeys(userId: string): Promise<void> {
  await withStore(OWN_KEY_PAIR_STORE, "readwrite", (store) => store.delete(userId));

  const db = await openDatabase();
  try {
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction(CHAT_KEYS_STORE, "readwrite");
      const store = tx.objectStore(CHAT_KEYS_STORE);
      const request = store.openCursor();

      request.onsuccess = () => {
        const cursor = request.result;
        if (!cursor) {
          return;
        }
        const record = cursor.value as ChatKeyRecord;
        if (record.userId === userId) {
          cursor.delete();
        }
        cursor.continue();
      };
      request.onerror = () => reject(request.error);
      tx.oncomplete = () => resolve();
      tx.onerror = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}
