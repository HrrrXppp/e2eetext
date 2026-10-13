import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LandingPage } from "@/app/LandingPage";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Site header</div>,
}));
vi.mock("@/components/layout/Footer", () => ({
  Footer: () => <div>Footer</div>,
}));

describe("LandingPage news", () => {
  it("lists the production launch first, above the project start entry", () => {
    render(<LandingPage />);
    const titles = screen
      .getAllByRole("heading", { level: 3 })
      .map((h) => h.textContent)
      .filter((t) => t === "E2EE Text is live in production" || t === "E2EE Text project started");
    expect(titles).toEqual(["E2EE Text is live in production", "E2EE Text project started"]);
    expect(screen.getByText("October 11, 2026")).toHaveAttribute("datetime", "2026-10-11");
  });
});
