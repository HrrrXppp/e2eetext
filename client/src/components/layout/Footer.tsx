import { donateUrl } from "@/lib/config";

export function Footer() {
  return (
    <footer className="site-foot">
      <p className="site-foot__support">
        E2EE Text will be sustained through{" "}
        <strong>advertisement</strong> and{" "}
        <a
          className="site-foot__donate"
          href={donateUrl}
          target="_blank"
          rel="noopener noreferrer"
        >
          <strong>donations</strong>
        </a>
      </p>
    </footer>
  );
}
