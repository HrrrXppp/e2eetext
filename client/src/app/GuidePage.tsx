import { AdSenseScript } from "@/components/ads/AdSenseScript";
import { Footer } from "@/components/layout/Footer";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { guides, type Guide } from "@/app/guides";

export function GuidePage({ guide }: { guide: Guide }) {
  const others = guides.filter((item) => item.path !== guide.path);

  return (
    <>
      <AdSenseScript />
      <SiteHeader />
      <main className="legal guide">
        <p className="guide__crumb">
          <a href="/guides">Guides</a>
        </p>
        <h1 className="legal__title">{guide.title}</h1>
        <p className="legal__updated">{guide.summary}</p>
        {guide.body}
        <h2>Other guides</h2>
        <ul className="guide-index">
          {others.map((item) => (
            <li key={item.path}>
              <a href={item.path}>
                <strong>{item.navLabel}</strong>
                <span>{item.summary}</span>
              </a>
            </li>
          ))}
        </ul>
      </main>
      <Footer />
    </>
  );
}
