import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GuidesPage } from "@/app/GuidesPage";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Header</div>,
}));
vi.mock("@/components/layout/Footer", () => ({
  Footer: () => <div>Footer</div>,
}));

describe("GuidesPage", () => {
  it("lists each guide and does not load the ad script", () => {
    render(<GuidesPage />);

    expect(screen.getByRole("heading", { level: 1, name: "Guides" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /What the server can and cannot read/ })).toHaveAttribute(
      "href",
      "/guides/what-the-server-can-see",
    );
    expect(screen.getByRole("link", { name: /How to confirm who you are talking to/ })).toHaveAttribute(
      "href",
      "/guides/who-you-are-talking-to",
    );
    expect(screen.getAllByRole("link")).toHaveLength(6);
    expect(document.querySelector('script[src*="adsbygoogle.js"]')).toBeNull();
  });
});
