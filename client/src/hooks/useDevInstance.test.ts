import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDevInstance } from "@/hooks/useDevInstance";
import { resetInstanceConfigCache } from "@/lib/instance";

describe("useDevInstance", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    resetInstanceConfigCache();
  });

  it("starts false before config loads", () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockReturnValue(new Promise(() => {})),
    );

    const { result } = renderHook(() => useDevInstance());
    expect(result.current).toBe(false);
  });

  it("shows the banner when instance.json is development", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ environment: "development" }),
      }),
    );

    const { result } = renderHook(() => useDevInstance());

    await waitFor(() => {
      expect(result.current).toBe(true);
    });
  });

  it("hides the banner when instance.json is production", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ environment: "production" }),
      }),
    );

    const { result } = renderHook(() => useDevInstance());

    await waitFor(() => {
      expect(result.current).toBe(false);
    });
  });
});
