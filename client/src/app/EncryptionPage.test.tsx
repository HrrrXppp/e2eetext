import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EncryptionPage } from "@/app/EncryptionPage";

vi.mock("@/components/layout/SiteHeader", () => ({
  SiteHeader: () => <div>Header</div>,
}));

describe("EncryptionPage", () => {
  it("explains the model in sections and ends with the technical appendix", () => {
    render(<EncryptionPage />);

    expect(screen.getByRole("heading", { level: 1, name: "How encryption works" })).toBeInTheDocument();
    const headings = screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);
    expect(headings).toEqual([
      "Overview",
      "Your identity keys",
      "Chat keys",
      "Key versions and rotation",
      "Identity backup",
      "Limits and honest caveats",
      "Technical appendix",
    ]);
  });

  it("shows the send and receive flow", () => {
    render(<EncryptionPage />);
    const steps = within(screen.getByRole("list", { name: "Message flow" })).getAllByRole("listitem");
    expect(steps).toHaveLength(3);
  });

  it("lists algorithm parameters in the appendix", () => {
    render(<EncryptionPage />);
    const table = screen.getByRole("table");
    expect(within(table).getByText(/ML-KEM-768 \+ X25519/)).toBeInTheDocument();
    expect(within(table).getByText(/N=2\^17, r=8, p=1/)).toBeInTheDocument();
  });

  it("shows the footer with an encryption link", () => {
    render(<EncryptionPage />);
    expect(screen.getByRole("link", { name: "Encryption" })).toHaveAttribute("href", "/encryption");
  });
});
