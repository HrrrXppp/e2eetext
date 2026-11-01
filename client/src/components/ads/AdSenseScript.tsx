import { useEffect } from "react";

const ADSENSE_CLIENT = "ca-pub-8559868695202171";

// Mounted only by the public pages (home, privacy, terms). The script is added
// when such a page mounts and removed when it unmounts, so the chat app and the
// roadmap never carry it. AdSense rejected the site for ads on pages that have
// no content of their own.
export function AdSenseScript() {
  useEffect(() => {
    const script = document.createElement("script");
    script.async = true;
    script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=${ADSENSE_CLIENT}`;
    script.crossOrigin = "anonymous";
    document.head.appendChild(script);
    return () => script.remove();
  }, []);

  return null;
}
