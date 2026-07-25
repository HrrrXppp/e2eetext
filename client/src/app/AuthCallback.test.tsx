import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthCallback } from "@/app/AuthCallback";
import { makeToken } from "@/test/makeToken";

const storeAuthSession = vi.fn();
const setPendingSignInName = vi.fn();

vi.mock("@/lib/auth", () => ({
  storeAuthSession: (...args: unknown[]) => storeAuthSession(...args),
}));

vi.mock("@/lib/signInPreferences", () => ({
  setPendingSignInName: (...args: unknown[]) => setPendingSignInName(...args),
}));

describe("AuthCallback", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    storeAuthSession.mockReset();
    setPendingSignInName.mockReset();
  });

  it("stores session and redirects when id token is present", async () => {
    const replace = vi.fn();
    vi.stubGlobal("location", {
      ...window.location,
      hash: "#id_token=test-id-token&access_token=access&refresh_token=refresh",
      replace,
    });

    render(<AuthCallback />);

    await waitFor(() => {
      expect(storeAuthSession).toHaveBeenCalledWith(
        {
          idToken: "test-id-token",
          accessToken: "access",
          refreshToken: "refresh",
        },
        undefined,
      );
    });
    expect(replace).toHaveBeenCalledWith("/chats");
  });

  it("passes the provider and one-time name through from the callback URL", async () => {
    const replace = vi.fn();
    vi.stubGlobal("location", {
      ...window.location,
      hash: "#id_token=test-id-token&provider=apple&name=Ada%20Appleseed",
      replace,
    });

    render(<AuthCallback />);

    await waitFor(() => {
      expect(storeAuthSession).toHaveBeenCalledWith(
        expect.objectContaining({ idToken: "test-id-token" }),
        "apple",
      );
    });
    expect(setPendingSignInName).toHaveBeenCalledWith("Ada Appleseed");
    expect(replace).toHaveBeenCalledWith("/chats");
  });

  it("clears any pending name when the callback carries none", async () => {
    const replace = vi.fn();
    vi.stubGlobal("location", {
      ...window.location,
      hash: "#id_token=test-id-token",
      replace,
    });

    render(<AuthCallback />);

    await waitFor(() => {
      expect(storeAuthSession).toHaveBeenCalled();
    });
    expect(setPendingSignInName).toHaveBeenCalledWith(undefined);
  });

  it("shows error when id token is missing", async () => {
    vi.stubGlobal("location", {
      ...window.location,
      hash: "#access_token=access",
      replace: vi.fn(),
    });

    render(<AuthCallback />);

    expect(await screen.findByText("Missing ID token in callback.")).toBeInTheDocument();
    expect(storeAuthSession).not.toHaveBeenCalled();
  });

  it("accepts a real jwt-shaped id token in the hash", async () => {
    const idToken = makeToken({
      sub: "google-subject-1",
      iss: "https://accounts.google.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    const replace = vi.fn();
    vi.stubGlobal("location", {
      ...window.location,
      hash: `#id_token=${encodeURIComponent(idToken)}`,
      replace,
    });

    render(<AuthCallback />);

    await waitFor(() => {
      expect(storeAuthSession).toHaveBeenCalled();
    });
    expect(replace).toHaveBeenCalledWith("/chats");
  });
});
