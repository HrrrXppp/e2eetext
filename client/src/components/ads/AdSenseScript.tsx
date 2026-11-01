import { useEffect } from "react";

const ADSENSE_CLIENT = "ca-pub-8559868695202171";
const SCRIPT_SELECTOR = `script[data-adsense-client="${ADSENSE_CLIENT}"]`;

function findScript(): HTMLScriptElement | null {
  const found = document.querySelector(SCRIPT_SELECTOR);
  return found instanceof HTMLScriptElement ? found : null;
}

// Loaded only by the public pages. The chat app must not pull this in:
// AdSense rejected the site for ads on pages that have no content of their own.
// The tag is removed when the last such page unmounts, so a client-side
// navigation into /chats does not leave it in the document.
export function AdSenseScript() {
  useEffect(() => {
    let script = findScript();
    if (!script) {
      script = document.createElement("script");
      script.async = true;
      script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${ADSENSE_CLIENT}`;
      script.crossOrigin = "anonymous";
      script.dataset.adsenseClient = ADSENSE_CLIENT;
      script.dataset.adsenseUsers = "0";
      document.head.appendChild(script);
    }
    const users = Number(script.dataset.adsenseUsers || "0") + 1;
    script.dataset.adsenseUsers = String(users);

    return () => {
      const current = findScript();
      if (!current) {
        return;
      }
      const remaining = Number(current.dataset.adsenseUsers || "1") - 1;
      if (remaining <= 0) {
        current.remove();
        return;
      }
      current.dataset.adsenseUsers = String(remaining);
    };
  }, []);

  return null;
}
