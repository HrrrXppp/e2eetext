import { guides } from "@/app/guides";
import { Footer } from "@/components/layout/Footer";
import { SiteHeader } from "@/components/layout/SiteHeader";

export function GuidesPage() {
  return (
    <>
      <SiteHeader />
      <main className="legal">
        <h1 className="legal__title">Guides</h1>
        <p className="legal__updated">
          How the messenger protects a conversation, and where that protection stops.
        </p>
        <p>
          Each page answers one question about encryption, disappearing messages, and how you
          know who is in a chat. The{" "}
          <a href="/encryption">encryption overview</a> is the short version, with the algorithm
          list.
        </p>
        <ul className="guide-index">
          {guides.map((guide) => (
            <li key={guide.path}>
              <a href={guide.path}>
                <strong>{guide.title}</strong>
                <span>{guide.summary}</span>
              </a>
            </li>
          ))}
        </ul>
      </main>
      <Footer />
    </>
  );
}
