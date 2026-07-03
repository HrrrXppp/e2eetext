import { useEffect, useRef, useState } from "react";
import { storeAuthSession } from "@/lib/auth";

export function AuthCallback() {
  const [error, setError] = useState<string | null>(null);
  const handledRef = useRef(false);

  useEffect(() => {
    if (handledRef.current) {
      return;
    }
    handledRef.current = true;

    const params = new URLSearchParams(window.location.hash.slice(1));
    const idToken = params.get("id_token");

    if (!idToken) {
      setError("Missing ID token in callback.");
      return;
    }

    storeAuthSession({
      idToken,
      accessToken: params.get("access_token") ?? undefined,
      refreshToken: params.get("refresh_token") ?? undefined,
    });

    window.location.replace("/chats");
  }, []);

  return (
    <main className="landing">
      <p>{error ?? "Signing you in..."}</p>
    </main>
  );
}
