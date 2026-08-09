function base64UrlEncodeUtf8(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

export function makeToken(payload: Record<string, unknown>): string {
  const header = base64UrlEncodeUtf8(JSON.stringify({ alg: "none", typ: "JWT" }));
  const body = base64UrlEncodeUtf8(JSON.stringify(payload));
  return `${header}.${body}.signature`;
}
