import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AdSenseScript } from "@/components/ads/AdSenseScript";

const SCRIPT_SELECTOR = 'script[src*="adsbygoogle.js"]';

describe("AdSenseScript", () => {
  it("adds the publisher script on mount", () => {
    const { unmount } = render(<AdSenseScript />);

    const scripts = document.querySelectorAll(SCRIPT_SELECTOR);
    expect(scripts).toHaveLength(1);
    expect(scripts[0]).toHaveAttribute(
      "src",
      "https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-8559868695202171",
    );
    expect(scripts[0]).toHaveAttribute("crossorigin", "anonymous");
    unmount();
  });

  it("removes the publisher script when the page unmounts", () => {
    const { unmount } = render(<AdSenseScript />);
    expect(document.querySelector(SCRIPT_SELECTOR)).not.toBeNull();

    unmount();
    expect(document.querySelector(SCRIPT_SELECTOR)).toBeNull();
  });

  it("keeps a single script when one page replaces another", () => {
    const first = render(<AdSenseScript />);
    first.unmount();
    const second = render(<AdSenseScript />);

    expect(document.querySelectorAll(SCRIPT_SELECTOR)).toHaveLength(1);
    second.unmount();
  });
});
