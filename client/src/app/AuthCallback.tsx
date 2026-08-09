import { useEffect, useRef, useState } from "react";
import { storeAuthSession } from "@/lib/auth";
import { setPendingSignInName } from "@/lib/signInPreferences";

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

    storeAuthSession(
      {
        idToken,
        accessToken: params.get("access_token") ?? undefined,
        refreshToken: params.get("refresh_token") ?? undefined,
      },
      params.get("provider") ?? undefined,
    );

    // Some providers (Apple) hand us the display name only once, out of
    // band from the ID token, right here — stash it so the next sign-in
    // registration call can pick it up after the navigation below.
    setPendingSignInName(params.get("name") ?? undefined);

    window.location.replace("/chats");
  }, []);

  return (
    <main className="landing">
      <p>{error ?? "Signing you in..."}</p>
    </main>
  );
}
