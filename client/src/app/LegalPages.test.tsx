import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrivacyPolicyPage } from "@/app/PrivacyPolicyPage";
import { TermsPage } from "@/app/TermsPage";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Header</div>,
}));

describe("legal pages", () => {
  it("renders the privacy policy with footer links", () => {
    render(<PrivacyPolicyPage />);
    expect(screen.getByRole("heading", { level: 1, name: "Privacy Policy" })).toBeInTheDocument();
    expect(screen.getByText(/never plaintext/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Terms of Service" })).toHaveAttribute("href", "/terms");
  });

  it("renders the terms of service with footer links", () => {
    render(<TermsPage />);
    expect(screen.getByRole("heading", { level: 1, name: "Terms of Service" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Privacy Policy" })).toHaveAttribute("href", "/privacy");
  });
});
