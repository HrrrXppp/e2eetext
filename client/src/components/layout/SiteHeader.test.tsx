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
});
