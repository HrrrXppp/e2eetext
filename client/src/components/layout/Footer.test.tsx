import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Footer } from "@/components/layout/Footer";

describe("Footer", () => {
  it("shows the sustainability message", () => {
    render(<Footer />);

    expect(
      screen.getByText(/E2EE Text will be sustained through/),
    ).toBeInTheDocument();
    expect(screen.getByText("advertisement")).toBeInTheDocument();
    expect(screen.getByText("donations")).toBeInTheDocument();
  });
});
