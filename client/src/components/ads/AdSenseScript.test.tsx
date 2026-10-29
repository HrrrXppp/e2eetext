import { render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AdSenseScript } from "@/components/ads/AdSenseScript";

const SCRIPT_SELECTOR = 'script[data-adsense-client="ca-pub-8559868695202171"]';

afterEach(() => {
  document.querySelector(SCRIPT_SELECTOR)?.remove();
});

describe("AdSenseScript", () => {
  it("adds the publisher script once", () => {
    const { rerender } = render(<AdSenseScript />);
    rerender(
      <>
        <AdSenseScript />
        <AdSenseScript />
      </>,
    );

    const scripts = document.querySelectorAll(SCRIPT_SELECTOR);
    expect(scripts).toHaveLength(1);
    expect(scripts[0]).toHaveAttribute(
      "src",
      "https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-8559868695202171",
    );
    expect(scripts[0]).toHaveAttribute("crossorigin", "anonymous");
  });
});
