import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SignInDialog } from "@/components/auth/SignInDialog";

const providers = [
  {
    id: "11111111-1111-1111-1111-111111111111",
    name: "Google",
    link: "https://accounts.google.com",
    slug: "google",
  },
];

describe("SignInDialog", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal("location", { href: "" });
  });

  it("stores skip-profile preference and starts provider login", () => {
    const onClose = vi.fn();

    render(
      <SignInDialog
        providers={providers}
        skipProfile={true}
        onSkipProfileChange={vi.fn()}
        onClose={onClose}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Sign in with Google" }));

    expect(localStorage.getItem("messenger_skip_profile_on_create")).toBe("1");
    expect(window.location.href).toBe("/api/v1/auth/google/login");
  });
});
