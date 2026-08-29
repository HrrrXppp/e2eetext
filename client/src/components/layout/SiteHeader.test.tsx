import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SiteHeader } from "@/components/layout/SiteHeader";

vi.mock("@/hooks/useAuth", () => ({
  useAuth: vi.fn(),
}));

import { useAuth } from "@/hooks/useAuth";

describe("SiteHeader OIDC providers", () => {
  it("shows the application version in the header", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });

    render(<SiteHeader />);

    expect(screen.getByTitle("Application version")).toHaveTextContent(/^version: \d+\.\d+\.\d+/);
  });

  it("shows a Chats link when the user is signed in", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: {
        id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
        subject: "google-subject-1",
        name: "Test User",
        provider: "google",
        oidcProviderId: "11111111-1111-1111-1111-111111111111",
      },
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });

    render(<SiteHeader />);

    expect(screen.getByRole("link", { name: "Chats" })).toHaveAttribute("href", "/chats");
    expect(screen.getByRole("button", { name: "Test User" })).toBeInTheDocument();
  });

  it("shows Add name when the stored user name is empty", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: {
        id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
        subject: "google-subject-1",
        provider: "google",
        oidcProviderId: "11111111-1111-1111-1111-111111111111",
      },
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });

    render(<SiteHeader />);

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add name" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" }),
    ).not.toBeInTheDocument();
  });

  it("opens a sign-in dialog with provider buttons", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      providers: [
        {
          id: "11111111-1111-1111-1111-111111111111",
          name: "Google",
          link: "https://accounts.google.com",
          slug: "google",
        },
      ],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });

    render(<SiteHeader />);

    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect(screen.getByRole("dialog", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Sign in with Google" })).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", {
        name: "Don't save name if user will be created",
      }),
    ).toBeInTheDocument();
  });

  it("shows a disabled sign-in state when no providers are configured", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
    });

    render(<SiteHeader />);

    expect(screen.getByText("Sign in")).toHaveAttribute("title", "No sign-in providers");
    expect(screen.queryByRole("dialog", { name: "Sign in" })).not.toBeInTheDocument();
  });

  it("wraps and truncates the header row instead of overlapping on narrow viewports (regression for #58)", () => {
    // Issue #58 (follow-up): .site-head__inner laid out the brand/version
    // group and the user/Chats/Sign-out group in a single non-wrapping flex
    // row with no min-width:0 or truncation on the user side. On phone-width
    // viewports the two groups' combined natural width exceeded the
    // available space, and since neither group could shrink or wrap, their
    // content visually overlapped (e.g. the version badge overlapping the
    // display name) instead of reflowing. This guards against that by
    // asserting the row can wrap, the user group can shrink below its
    // content size, and the display name truncates instead of overflowing.
    const cssPath = path.resolve(
      path.dirname(fileURLToPath(import.meta.url)),
      "../../styles/index.css",
    );
    const css = readFileSync(cssPath, "utf8");

    const innerMatch = css.match(/\.site-head__inner\s*\{([^}]*)\}/);
    expect(innerMatch).not.toBeNull();
    expect(innerMatch![1]).toMatch(/flex-wrap:\s*wrap/);

    const userMatch = css.match(/\.site-head__user\s*\{([^}]*)\}/);
    expect(userMatch).not.toBeNull();
    expect(userMatch![1]).toMatch(/min-width:\s*0/);

    const nameMatch = css.match(/\.site-head__user-name\s*\{([^}]*)\}/);
    expect(nameMatch).not.toBeNull();
    expect(nameMatch![1]).toMatch(/overflow:\s*hidden/);
    expect(nameMatch![1]).toMatch(/text-overflow:\s*ellipsis/);
  });
});
