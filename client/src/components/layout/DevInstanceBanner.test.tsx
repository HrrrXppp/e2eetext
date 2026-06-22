import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DevInstanceBanner } from "@/components/layout/DevInstanceBanner";
import { DEV_INSTANCE_BANNER_TEXT } from "@/lib/instance";

vi.mock("@/hooks/useDevInstance", () => ({
  useDevInstance: vi.fn(),
}));

import { useDevInstance } from "@/hooks/useDevInstance";

describe("DevInstanceBanner", () => {
  afterEach(() => {
    document.body.classList.remove("has-dev-instance-banner");
    vi.mocked(useDevInstance).mockReset();
  });

  it("renders the banner on development instances", () => {
    vi.mocked(useDevInstance).mockReturnValue(true);

    render(<DevInstanceBanner />);

    expect(screen.getByRole("status")).toHaveTextContent(DEV_INSTANCE_BANNER_TEXT);
    expect(document.body.classList.contains("has-dev-instance-banner")).toBe(true);
  });

  it("renders nothing on production instances", () => {
    vi.mocked(useDevInstance).mockReturnValue(false);

    const { container } = render(<DevInstanceBanner />);

    expect(container).toBeEmptyDOMElement();
    expect(document.body.classList.contains("has-dev-instance-banner")).toBe(false);
  });

  it("updates body class when dev state changes", async () => {
    vi.mocked(useDevInstance).mockReturnValue(false);

    const { rerender } = render(<DevInstanceBanner />);
    expect(document.body.classList.contains("has-dev-instance-banner")).toBe(false);

    vi.mocked(useDevInstance).mockReturnValue(true);
    rerender(<DevInstanceBanner />);

    await waitFor(() => {
      expect(document.body.classList.contains("has-dev-instance-banner")).toBe(true);
    });
  });
});
