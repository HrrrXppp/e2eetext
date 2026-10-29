import type { ReactNode } from "react";
import { AdSenseScript } from "@/components/ads/AdSenseScript";
import { Footer } from "@/components/layout/Footer";
import { SiteHeader } from "@/components/layout/SiteHeader";

export function LegalPage({
  title,
  updated,
  children,
}: {
  title: string;
  updated: string;
  children: ReactNode;
}) {
  return (
    <>
      <AdSenseScript />
      <SiteHeader />
      <main className="legal">
        <h1 className="legal__title">{title}</h1>
        <p className="legal__updated">Last updated: {updated}</p>
        {children}
      </main>
      <Footer />
    </>
  );
}
