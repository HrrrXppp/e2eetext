import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ContactPage } from "@/app/ContactPage";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Header</div>,
}));

describe("ContactPage", () => {
  it("shows headings, both mailto links and the GitHub issues link", () => {
    render(<ContactPage />);

    expect(screen.getByRole("heading", { level: 1, name: "Contact" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Help and support" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Marketing and cooperation" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "help@e2eetext.com" })).toHaveAttribute(
      "href",
      "mailto:help@e2eetext.com",
    );
    expect(screen.getByRole("link", { name: "marketing@e2eetext.com" })).toHaveAttribute(
      "href",
      "mailto:marketing@e2eetext.com",
    );
    const gh = screen.getByRole("link", { name: "GitHub issues" });
    expect(gh).toHaveAttribute("href", "https://github.com/HrrrXppp/e2eetext/issues");
    expect(gh).toHaveAttribute("target", "_blank");
  });

  it("copies an address to the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
    render(<ContactPage />);

    fireEvent.click(screen.getByRole("button", { name: "Copy help@e2eetext.com" }));

    expect(writeText).toHaveBeenCalledWith("help@e2eetext.com");
    expect(await screen.findByText("Copied")).toBeInTheDocument();
    vi.unstubAllGlobals();
  });
});
