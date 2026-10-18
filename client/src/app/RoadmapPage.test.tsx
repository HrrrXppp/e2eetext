import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RoadmapPage } from "@/app/RoadmapPage";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Header</div>,
}));

describe("RoadmapPage", () => {
  it("lists the four roadmap points in order with quarter, status and description", () => {
    render(<RoadmapPage />);

    const items = within(screen.getByRole("list")).getAllByRole("listitem");
    expect(items).toHaveLength(4);
    const expected = [
      ["Base improvement", "Q4 2026", "In progress"],
      ["Separated services with custom LB for good scale", "Q4 2026", "Planned"],
      ["Audio calls", "Q1 2027", "Planned"],
      ["Video calls", "Q1 2027", "Planned"],
    ];
    expected.forEach(([title, quarter, status], i) => {
      const item = within(items[i]);
      expect(item.getByRole("heading", { level: 2, name: title })).toBeInTheDocument();
      expect(item.getByText(quarter)).toBeInTheDocument();
      expect(item.getByText(status)).toBeInTheDocument();
    });
  });

  it("shows the footer with a roadmap link", () => {
    render(<RoadmapPage />);
    expect(screen.getByRole("link", { name: "Roadmap" })).toHaveAttribute("href", "/roadmap");
  });
});
