// Public client-side settings. Override at build time via Vite env vars
// (e.g. `VITE_DONATE_URL=... npm run build`).
const DEFAULT_DONATE_URL = "https://donate.stripe.com/8x214g0wr4bWe1RcXZ9R600";

export const donateUrl: string =
  import.meta.env.VITE_DONATE_URL || DEFAULT_DONATE_URL;
