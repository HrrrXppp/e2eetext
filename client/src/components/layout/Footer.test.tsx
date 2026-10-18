import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Footer } from "@/components/layout/Footer";

describe("Footer", () => {
  it("keeps the sustain text and links donations so it opens the Stripe page in a new tab", () => {
    render(<Footer />);

    expect(screen.getByText(/E2EE Text will be sustained through/)).toBeInTheDocument();
    expect(screen.getByText("advertisement")).toBeInTheDocument();

    const link = screen.getByRole("link", { name: "donations" });
    expect(link).toHaveAttribute(
      "href",
      "https://donate.stripe.com/8x214g0wr4bWe1RcXZ9R600",
    );
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("links privacy, terms and contact and shows the owner copyright", () => {
    render(<Footer />);

    expect(screen.getByRole("link", { name: "Privacy Policy" })).toHaveAttribute("href", "/privacy");
    expect(screen.getByRole("link", { name: "Terms of Service" })).toHaveAttribute("href", "/terms");
    expect(screen.getByRole("link", { name: "Roadmap" })).toHaveAttribute("href", "/roadmap");
    expect(screen.getByRole("link", { name: "Contact" })).toHaveAttribute(
      "href",
      "https://github.com/HrrrXppp/e2eetext/issues",
    );
    expect(screen.getByText(/2026 e2eetext/)).toBeInTheDocument();
  });
});
