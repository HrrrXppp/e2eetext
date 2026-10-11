import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Footer } from "@/components/layout/Footer";

describe("Footer", () => {
  it("renders a Donate link that opens the Stripe page in a new tab", () => {
    render(<Footer />);

    const link = screen.getByRole("link", { name: "Donate" });
    expect(link).toHaveAttribute(
      "href",
      "https://donate.stripe.com/8x214g0wr4bWe1RcXZ9R600",
    );
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });
});
