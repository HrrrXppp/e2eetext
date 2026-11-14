import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Main } from "@/app/Main";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Site header</div>,
}));
vi.mock("@/components/layout/Footer", () => ({
  Footer: () => <div>Footer</div>,
}));

describe("Main news", () => {
  it("announces the roadmap in a keyboard-focusable scrollable list", () => {
    render(<Main />);
    expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();
    expect(screen.queryByText("Available now")).not.toBeInTheDocument();
    expect(screen.getByText("October 18, 2026")).toHaveAttribute("datetime", "2026-10-18");
    expect(screen.getByText(/encrypted in your browser/)).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3, name: "Roadmap published" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "View the roadmap" })).toHaveAttribute("href", "/roadmap");
    expect(screen.queryByRole("heading", { name: "Guides" })).not.toBeInTheDocument();
    expect(document.querySelector('script[src*="adsbygoogle.js"]')).toBeNull();
    const list = screen.getByRole("list", { name: "News items" });
    expect(list).toHaveAttribute("tabindex", "0");
    expect(list).toHaveClass("main__news-list");
  });

  it("lists the production launch first, above the project start entry", () => {
    render(<Main />);
    const titles = screen
      .getAllByRole("heading", { level: 3 })
      .map((h) => h.textContent)
      .filter((t) => t === "E2EE Text is live in production" || t === "E2EE Text project started");
    expect(titles).toEqual(["E2EE Text is live in production", "E2EE Text project started"]);
    expect(screen.getByText("October 11, 2026")).toHaveAttribute("datetime", "2026-10-11");
  });
});
