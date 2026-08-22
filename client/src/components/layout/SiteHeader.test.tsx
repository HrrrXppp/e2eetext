import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SiteHeader } from "@/components/layout/SiteHeader";

vi.mock("@/hooks/useAuth", () => ({
  useAuth: vi.fn(),
}));

vi.mock("@/lib/e2ee/storage", () => ({
  exportStoredIdentityBackup: vi.fn(),
  fetchIdentityPublicKey: vi.fn(),
  loadStoredIdentity: vi.fn().mockResolvedValue(null),
  saveStoredIdentity: vi.fn(),
  uploadIdentityKey: vi.fn(),
}));

vi.mock("@/lib/e2ee/crypto", () => ({
  importIdentityBackup: vi.fn(),
}));

import { useAuth } from "@/hooks/useAuth";

const SIGNED_IN_USER = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  subject: "google-subject-1",
  name: "Test User",
  provider: "google",
  oidcProviderId: "11111111-1111-1111-1111-111111111111",
};

describe("SiteHeader OIDC providers", () => {
  beforeEach(() => {
    vi.mocked(useAuth).mockReset();
  });

  it("shows the application version in the header", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
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
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });

    render(<SiteHeader />);

    expect(screen.getByText("Sign in")).toHaveAttribute("title", "No sign-in providers");
    expect(screen.queryByRole("dialog", { name: "Sign in" })).not.toBeInTheDocument();
  });

  it("opens the backup dialog from the Back up key button", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: SIGNED_IN_USER,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });

    render(<SiteHeader />);

    fireEvent.click(screen.getByRole("button", { name: "Back up key" }));

    expect(screen.getByText("Back up your private key")).toBeInTheDocument();
  });

  it("opens the restore dialog from the Restore key button", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: SIGNED_IN_USER,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });

    render(<SiteHeader />);

    fireEvent.click(screen.getByRole("button", { name: "Restore key" }));

    expect(screen.getByText("Restore your private key")).toBeInTheDocument();
  });

  it("shows the one-time identity backup prompt right after a new key is generated", () => {
    const acknowledgeIdentityBackup = vi.fn();
    vi.mocked(useAuth).mockReturnValue({
      user: SIGNED_IN_USER,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: true,
      acknowledgeIdentityBackup,
    });

    render(<SiteHeader />);

    expect(screen.getByText("Save a backup of your private key")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Skip for now" }));
    expect(acknowledgeIdentityBackup).toHaveBeenCalled();
  });

  it("does not show the identity backup prompt once acknowledged", () => {
    vi.mocked(useAuth).mockReturnValue({
      user: SIGNED_IN_USER,
      providers: [],
      loading: false,
      signOut: vi.fn(),
      setDisplayName: vi.fn(),
      justCreatedIdentity: false,
      acknowledgeIdentityBackup: vi.fn(),
    });

    render(<SiteHeader />);

    expect(screen.queryByText("Save a backup of your private key")).not.toBeInTheDocument();
  });
});
