const DONATE_URL = "https://donate.stripe.com/8x214g0wr4bWe1RcXZ9R600";

export function Footer() {
  return (
    <footer className="site-foot">
      <p className="site-foot__support">
        <a
          className="site-foot__donate"
          href={DONATE_URL}
          target="_blank"
          rel="noopener noreferrer"
        >
          Donate
        </a>
      </p>
    </footer>
  );
}
