import { useEffect } from "react";

const ADSENSE_CLIENT = "ca-pub-8559868695202171";

// Mounted only by /encryption and the individual guide articles. The script is
// added when such a page mounts and removed when it unmounts, so the home
// page, the /guides index, the legal pages, the chat app, and the roadmap
// never carry it. AdSense rejected the site for ads on screens without
// publisher content.
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
