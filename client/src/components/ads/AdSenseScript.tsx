import { useEffect } from "react";

const ADSENSE_CLIENT = "ca-pub-8559868695202171";
const SCRIPT_SELECTOR = `script[data-adsense-client="${ADSENSE_CLIENT}"]`;

// Loaded only by the public pages. The chat app must not pull this in:
// AdSense rejected the site for ads on pages that have no content of their own.
export function AdSenseScript() {
  useEffect(() => {
    if (document.querySelector(SCRIPT_SELECTOR)) {
      return;
    }
    const script = document.createElement("script");
    script.async = true;
    script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${ADSENSE_CLIENT}`;
    script.crossOrigin = "anonymous";
    script.dataset.adsenseClient = ADSENSE_CLIENT;
    document.head.appendChild(script);
  }, []);

  return null;
}
