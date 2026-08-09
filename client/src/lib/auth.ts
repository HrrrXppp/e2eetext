import { API_V1 } from "@/lib/api";
import { getTokenExpiration, isTokenExpired, parseIdToken } from "@/lib/idToken";

const ID_TOKEN_KEY = "messenger_id_token";
const REFRESH_TOKEN_KEY = "messenger_refresh_token";
const ACCESS_TOKEN_KEY = "messenger_access_token";
const AUTH_PROVIDER_KEY = "messenger_auth_provider";
const TOKEN_EXPIRES_AT_KEY = "messenger_token_expires_at";

export const TOKEN_REFRESHED_EVENT = "messenger:token-refreshed";

const REFRESH_SKEW_SECONDS = 300;
const REFRESH_CHECK_INTERVAL_MS = 60_000;

export type AuthTokens = {
  idToken: string;
  accessToken?: string;
  refreshToken?: string;
};

let refreshInFlight: Promise<boolean> | null = null;

export function getIdToken(): string | null {
  return localStorage.getItem(ID_TOKEN_KEY);
}

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function getAuthProvider(): string | null {
  return localStorage.getItem(AUTH_PROVIDER_KEY);
}

export function setIdToken(token: string): void {
  localStorage.setItem(ID_TOKEN_KEY, token);
}

export function storeAuthSession(tokens: AuthTokens, provider?: string): void {
  setIdToken(tokens.idToken);

  if (tokens.accessToken) {
    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
  }

  if (tokens.refreshToken) {
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
  }

  const expiresAt = getTokenExpiration(tokens.idToken);
  if (expiresAt) {
    localStorage.setItem(TOKEN_EXPIRES_AT_KEY, String(expiresAt));
  }

  const resolvedProvider = provider ?? parseIdToken(tokens.idToken)?.provider;
  if (resolvedProvider) {
    localStorage.setItem(AUTH_PROVIDER_KEY, resolvedProvider);
  }
}

export function clearIdToken(): void {
  localStorage.removeItem(ID_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(AUTH_PROVIDER_KEY);
  localStorage.removeItem(TOKEN_EXPIRES_AT_KEY);
}

export function getBearerToken(): string | null {
  const idToken = getIdToken();
  if (idToken && !isTokenExpired(idToken, 0)) {
    return idToken;
  }

  return getAccessToken() ?? idToken;
}

export function getWebSocketIdToken(): string | null {
  const idToken = getIdToken();
  if (!idToken || isTokenExpired(idToken, 0)) {
    return null;
  }

  return idToken;
}

export function authHeaders(): HeadersInit {
  const token = getBearerToken();
  if (!token) {
    return {};
  }

  return {
    Authorization: `Bearer ${token}`,
  };
}

function isSessionExpired(skewSeconds: number): boolean {
  const expiresAtRaw = localStorage.getItem(TOKEN_EXPIRES_AT_KEY);
  if (!expiresAtRaw) {
    const idToken = getIdToken();
    if (!idToken) {
      return true;
    }
    return isTokenExpired(idToken, skewSeconds);
  }

  const expiresAt = Number(expiresAtRaw);
  if (!Number.isFinite(expiresAt)) {
    return true;
  }

  return Date.now() / 1000 >= expiresAt - skewSeconds;
}

export function hasUsableSession(): boolean {
  return getBearerToken() !== null && !isSessionExpired(0);
}

export async function refreshAuthTokensIfNeeded(options?: {
  skewSeconds?: number;
}): Promise<boolean> {
  if (!getIdToken() && !getAccessToken()) {
    return false;
  }

  const skewSeconds = options?.skewSeconds ?? REFRESH_SKEW_SECONDS;
  if (!isSessionExpired(skewSeconds)) {
    return true;
  }

  if (refreshInFlight) {
    return refreshInFlight;
  }

  refreshInFlight = performTokenRefresh().finally(() => {
    refreshInFlight = null;
  });

  return refreshInFlight;
}

async function performTokenRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  const provider = getAuthProvider();
  if (!refreshToken || !provider) {
    return hasUsableSession();
  }

  const response = await fetch(`${API_V1}/auth/refresh`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      provider,
      refreshToken,
    }),
  });

  if (!response.ok) {
    return hasUsableSession();
  }

  const data = (await response.json()) as {
    idToken: string;
    accessToken?: string;
    refreshToken?: string;
  };

  storeAuthSession(
    {
      idToken: data.idToken,
      accessToken: data.accessToken,
      refreshToken: data.refreshToken ?? refreshToken,
    },
    provider,
  );

  window.dispatchEvent(new Event(TOKEN_REFRESHED_EVENT));
  return true;
}

export function startTokenRefreshLoop(onRefreshFailed: () => void): () => void {
  async function checkToken() {
    if (!getIdToken() && !getAccessToken()) {
      return;
    }

    const refreshed = await refreshAuthTokensIfNeeded();
    if (!refreshed || isSessionExpired(0)) {
      onRefreshFailed();
    }
  }

  void checkToken();

  const interval = window.setInterval(() => {
    void checkToken();
  }, REFRESH_CHECK_INTERVAL_MS);

  function handleVisibilityChange() {
    if (document.visibilityState === "visible") {
      void checkToken();
    }
  }

  document.addEventListener("visibilitychange", handleVisibilityChange);

  return () => {
    clearInterval(interval);
    document.removeEventListener("visibilitychange", handleVisibilityChange);
  };
}
