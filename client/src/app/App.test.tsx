import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { App } from "@/app/App";

vi.mock("@/app/AuthCallback", () => ({
  AuthCallback: () => <div>Auth callback page</div>,
}));
vi.mock("@/app/ChatsPage", () => ({
  ChatsPage: () => <div>Chats page</div>,
}));
vi.mock("@/app/LandingPage", () => ({
  LandingPage: () => <div>Landing page</div>,
}));

vi.mock("@/app/PrivacyPolicyPage", () => ({
  PrivacyPolicyPage: () => <div>Privacy page</div>,
}));
vi.mock("@/app/TermsPage", () => ({
  TermsPage: () => <div>Terms page</div>,
}));

vi.mock("@/app/RoadmapPage", () => ({
  RoadmapPage: () => <div>Roadmap page</div>,
}));

describe("App", () => {
  it("routes oauth callback path", () => {
    vi.stubGlobal("location", { ...window.location, pathname: "/oauth/callback" });
    render(<App />);
    expect(screen.getByText("Auth callback page")).toBeInTheDocument();
  });

  it("routes chats path", () => {
    vi.stubGlobal("location", { ...window.location, pathname: "/chats" });
    render(<App />);
    expect(screen.getByText("Chats page")).toBeInTheDocument();
  });

  it("routes landing page for other paths", () => {
    vi.stubGlobal("location", { ...window.location, pathname: "/" });
    render(<App />);
    expect(screen.getByText("Landing page")).toBeInTheDocument();
  });

  it("routes privacy and terms paths", () => {
    vi.stubGlobal("location", { ...window.location, pathname: "/privacy" });
    const { unmount } = render(<App />);
    expect(screen.getByText("Privacy page")).toBeInTheDocument();
    unmount();
    vi.stubGlobal("location", { ...window.location, pathname: "/terms" });
    render(<App />);
    expect(screen.getByText("Terms page")).toBeInTheDocument();
  });

  it("routes roadmap path", () => {
    vi.stubGlobal("location", { ...window.location, pathname: "/roadmap" });
    render(<App />);
    expect(screen.getByText("Roadmap page")).toBeInTheDocument();
  });
});
