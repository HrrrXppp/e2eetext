import { API_V1 } from "@/lib/api";
import { authHeaders } from "@/lib/auth";
import { encodeBase64 } from "@/lib/bytes";
import { generateKemKeyPair } from "@/lib/crypto";
import type { IdTokenClaims } from "@/lib/idToken";
import { saveOwnKeyPair } from "@/lib/keyStore";

export type User = {
  id: string;
  oidcProviderId: string;
  subject: string;
  name?: string;
  kemPublicKey: string;
  createdAt: string;
  updatedAt: string;
};

export function userDisplayName(name?: string): string {
  const trimmed = name?.trim();
  if (trimmed) {
    return trimmed;
  }
  return "Add name";
}

export function userIdFallbackLabel(userId: string): string {
  const [nodeId, localId] = userId.split("/");
  if (nodeId && localId) {
    return `${idLeadingSegment(nodeId)}...${idTrailingSegment(localId)}`;
  }
  return `${idLeadingSegment(userId)}...${idTrailingSegment(userId)}`;
}

function idLeadingSegment(id: string): string {
  const [first] = id.split("-");
  return first || id;
}

function idTrailingSegment(id: string): string {
  const parts = id.split("-");
  return parts[parts.length - 1] || id;
}

export function userLabel(userName: string | undefined, userId: string): string {
  const trimmed = userName?.trim();
  if (trimmed) {
    return trimmed;
  }
  return userIdFallbackLabel(userId);
}

export type UserSearchInput = {
  query: string;
};

const MIN_NAME_SEARCH_LENGTH = 3;

export function canSearchUsers(query: string): boolean {
  return [...query.trim()].length >= MIN_NAME_SEARCH_LENGTH;
}

export async function searchUsers(query: string): Promise<User[]> {
  const q = query.trim();
  if (!canSearchUsers(q)) {
    return [];
  }

  const params = new URLSearchParams({ q });
  const response = await fetch(`${API_V1}/search?${params.toString()}`, {
    headers: authHeaders(),
  });

  if (!response.ok) {
    if (response.status === 404 || response.status === 400) {
      return [];
    }
    throw new Error("failed to search users");
  }

  return response.json() as Promise<User[]>;
}

type UserFilter = {
  subject: string;
  oidcProviderId: string;
};

type CreateUserOptions = {
  skipProfile?: boolean;
  kemPublicKey: string;
};

export async function fetchUsers(filter: UserFilter): Promise<User[]> {
  const params = new URLSearchParams({
    subject: filter.subject,
    oidc_provider_id: filter.oidcProviderId,
  });

  const response = await fetch(`${API_V1}/user?${params.toString()}`, {
    headers: authHeaders(),
  });

  if (!response.ok) {
    throw new Error("failed to fetch users");
  }

  return response.json() as Promise<User[]>;
}

export async function updateUserName(userId: string, name: string): Promise<User> {
  const response = await fetch(`${API_V1}/user/${userId}`, {
    method: "PATCH",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name }),
  });

  if (response.status === 403) {
    throw new Error("user access denied");
  }

  if (!response.ok) {
    throw new Error("failed to update user name");
  }

  return response.json() as Promise<User>;
}

export async function createUser(options: CreateUserOptions): Promise<User> {
  const body: Record<string, unknown> = { kem_public_key: options.kemPublicKey };
  if (options.skipProfile) {
    body.skip_profile = true;
  }

  const response = await fetch(`${API_V1}/user`, {
    method: "POST",
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (response.status === 409) {
    throw new Error("user already exists");
  }

  if (!response.ok) {
    throw new Error("failed to create user");
  }

  return response.json() as Promise<User>;
}

type EnsureUserOptions = {
  skipProfile?: boolean;
};

const ensureUserInflight = new Map<string, Promise<User>>();

function ensureUserKey(
  claims: IdTokenClaims,
  oidcProviderId: string,
  options?: EnsureUserOptions,
): string {
  return `${oidcProviderId}:${claims.subject}:${options?.skipProfile ? "1" : "0"}`;
}

export async function ensureUserRegistered(
  claims: IdTokenClaims,
  oidcProviderId: string,
  options?: EnsureUserOptions,
): Promise<User> {
  const key = ensureUserKey(claims, oidcProviderId, options);
  const inflight = ensureUserInflight.get(key);
  if (inflight) {
    return inflight;
  }

  const promise = ensureUserRegisteredOnce(claims, oidcProviderId, options).finally(() => {
    ensureUserInflight.delete(key);
  });
  ensureUserInflight.set(key, promise);
  return promise;
}

async function ensureUserRegisteredOnce(
  claims: IdTokenClaims,
  oidcProviderId: string,
  options?: EnsureUserOptions,
): Promise<User> {
  const filter = {
    subject: claims.subject,
    oidcProviderId,
  };

  const users = await fetchUsers(filter);
  if (users.length > 0) {
    return users[0];
  }

  const keyPair = generateKemKeyPair();

  try {
    const created = await createUser({
      skipProfile: options?.skipProfile,
      kemPublicKey: encodeBase64(keyPair.publicKey),
    });
    await saveOwnKeyPair(created.id, keyPair);
    return created;
  } catch (error) {
    if (error instanceof Error && error.message === "user already exists") {
      const existingUsers = await fetchUsers(filter);
      if (existingUsers.length > 0) {
        return existingUsers[0];
      }
    }
    throw error;
  }
}
