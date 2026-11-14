import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GuidePage } from "@/app/GuidePage";
import { guideByPath, guides } from "@/app/guides";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Header</div>,
}));
vi.mock("@/components/layout/Footer", () => ({
  Footer: () => <div>Footer</div>,
}));

const SCRIPT_SELECTOR = 'script[src*="adsbygoogle.js"]';

describe("GuidePage", () => {
  it("answers one question and links the other guides", () => {
    const guide = guideByPath("/guides/disappearing-messages");
    if (!guide) {
      throw new Error("missing disappearing-messages guide");
    }
    render(<GuidePage guide={guide} />);

    expect(screen.getByRole("heading", { level: 1, name: guide.title })).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 2, name: "When a message expires" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Guides" })).toHaveAttribute("href", "/guides");
    expect(screen.queryByRole("link", { name: /Disappearing messages/ })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Chat keys/ })).toHaveAttribute("href", "/guides/chat-keys");
    expect(document.querySelectorAll(SCRIPT_SELECTOR)).toHaveLength(1);
  });

  it("has a page for every guide path", () => {
    expect(guides.map((guide) => guide.path)).toEqual([
      "/guides/what-the-server-can-see",
      "/guides/chat-keys",
      "/guides/disappearing-messages",
      "/guides/identity-backup",
      "/guides/who-you-are-talking-to",
    ]);
  });
});
