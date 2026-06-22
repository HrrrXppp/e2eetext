function decodeBase64UrlJson<T>(segment: string): T {
  const base64 = segment.replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(base64);
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
  const json = new TextDecoder("utf-8").decode(bytes);
  return JSON.parse(json) as T;
}

export type IdTokenClaims = {
  subject: string;
  name?: string;
  provider: string;
  expiresAt?: number;
};

export function getTokenExpiration(token: string): number | null {
  const parts = token.split(".");
  if (parts.length !== 3) {
    return null;
  }

  try {
    const payload = decodeBase64UrlJson<{ exp?: number }>(parts[1]);
    return typeof payload.exp === "number" ? payload.exp : null;
  } catch {
    return null;
  }
}

export function isTokenExpired(token: string, skewSeconds = 0): boolean {
  const expiresAt = getTokenExpiration(token);
  if (!expiresAt) {
    return false;
  }

  return Date.now() / 1000 >= expiresAt - skewSeconds;
}

export function parseIdToken(token: string): IdTokenClaims | null {
  const parts = token.split(".");
  if (parts.length !== 3) {
    return null;
  }

  try {
    const payload = decodeBase64UrlJson<{
      sub?: string;
      name?: string;
      iss?: string;
    }>(parts[1]);

    if (!payload.sub) {
      return null;
    }

    return {
      subject: payload.sub,
      name: payload.name,
      provider: payload.iss?.includes("google.com") ? "google" : "oidc",
      expiresAt: getTokenExpiration(token) ?? undefined,
    };
  } catch {
    return null;
  }
}
