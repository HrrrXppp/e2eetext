import type { ReactNode } from "react";
import { SiteHeader } from "@/components/layout/SiteHeader";

const features = [
  {
    icon: "web",
    title: "Web-based",
    description:
      "Use E2EE Text from any modern browser—no install required, always up to date.",
  },
  {
    icon: "lock",
    title: "End-to-end encrypted",
    description:
      "Only you and the people you talk to can read your messages. Not even we can.",
  },
  {
    icon: "quantum",
    title: "Post-quantum cryptography",
    description:
      "Built with quantum-resistant encryption so your conversations stay private tomorrow.",
  },
] as const;

const news = [
  {
    date: "May 31, 2026",
    dateTime: "2026-05-31",
    title: "E2EE Text project started",
    body:
      "We have officially kicked off E2EE Text — End-To-End Encryption Text. The landing page is live while we build a web-based messenger with end-to-end encryption and post-quantum cryptography. Stay tuned for updates as development progresses.",
  },
] as const;

function SectionHeading({ id, children }: { id: string; children: ReactNode }) {
  return (
    <h2 id={id} className="landing__section-title">
      <span className="landing__section-line" aria-hidden="true" />
      {children}
    </h2>
  );
}

function FeatureIcon({ name }: { name: (typeof features)[number]["icon"] }) {
  return (
    <span className={`feature-icon feature-icon--${name}`} aria-hidden="true">
      {name === "web" && (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <circle cx="12" cy="12" r="9" />
          <path d="M3 12h18M12 3c2.5 3 2.5 15 0 18M12 3c-2.5 3-2.5 15 0 18" />
        </svg>
      )}
      {name === "lock" && (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <rect x="5" y="11" width="14" height="10" rx="2" />
          <path d="M8 11V8a4 4 0 0 1 8 0v3" />
        </svg>
      )}
      {name === "quantum" && (
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">
          <circle cx="12" cy="12" r="2" />
          <ellipse cx="12" cy="12" rx="9" ry="4" />
          <ellipse cx="12" cy="12" rx="9" ry="4" transform="rotate(60 12 12)" />
          <ellipse cx="12" cy="12" rx="9" ry="4" transform="rotate(120 12 12)" />
        </svg>
      )}
    </span>
  );
}

export function LandingPage() {
  return (
    <>
      <SiteHeader />
      <div className="landing">
        <div className="landing__backdrop" aria-hidden="true">
          <div className="landing__orb landing__orb--1" />
          <div className="landing__orb landing__orb--2" />
          <div className="landing__grid" />
        </div>

        <div className="landing__hero">
          <p className="landing__eyebrow">
            <span className="landing__eyebrow-dot" />
            Coming soon
          </p>
          <h1 className="landing__title">
            <span className="landing__title-accent">E2EE</span> Text
          </h1>
          <p className="landing__tagline">End-To-End Encryption Text</p>
          <p className="landing__lead">
            A secure, web-based messenger designed for private conversations in an
            uncertain future.
          </p>
        </div>

        <section className="landing__panel landing__about" aria-labelledby="about-heading">
          <SectionHeading id="about-heading">What is E2EE Text?</SectionHeading>
          <p className="landing__description">
            E2EE Text (End-To-End Encryption Text) is a web-based chat platform
            that protects every message with end-to-end encryption and
            post-quantum cryptographic methods. Your conversations stay between
            you and your contacts—private by design, resilient against
            tomorrow&apos;s threats.
          </p>
          <ul className="landing__highlights">
            <li className="landing__highlight">
              <span className="landing__highlight-icon" aria-hidden="true">✓</span>
              We don&apos;t keep any user data
            </li>
            <li className="landing__highlight">
              <span className="landing__highlight-icon" aria-hidden="true">✓</span>
              Disappearing messages by design
            </li>
          </ul>
        </section>

        <section className="landing__panel landing__news" aria-labelledby="news-heading">
          <SectionHeading id="news-heading">News</SectionHeading>
          <ul className="landing__news-list">
            {news.map((item) => (
              <li key={item.title} className="landing__news-item">
                <time className="landing__news-date" dateTime={item.dateTime}>
                  {item.date}
                </time>
                <h3 className="landing__news-title">{item.title}</h3>
                <p className="landing__news-body">{item.body}</p>
              </li>
            ))}
          </ul>
        </section>

        <section className="landing__features" aria-labelledby="features-heading">
          <h2 id="features-heading" className="landing__section-title landing__section-title--center">
            <span className="landing__section-line" aria-hidden="true" />
            Built for real privacy
            <span className="landing__section-line" aria-hidden="true" />
          </h2>
          <ul className="landing__feature-list">
            {features.map((feature) => (
              <li key={feature.title} className="landing__feature">
                <FeatureIcon name={feature.icon} />
                <h3>{feature.title}</h3>
                <p>{feature.description}</p>
              </li>
            ))}
          </ul>
        </section>

        <footer className="landing__footer">
          <p className="landing__support">
            E2EE Text will be sustained through{" "}
            <strong>advertisement</strong> and <strong>donations</strong>
            <span className="landing__soon"> — coming soon</span>
          </p>
        </footer>
      </div>
    </>
  );
}
