import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/notifications", () => ({
  isNotificationSupported: vi.fn(),
  getNotificationPermission: vi.fn(),
  requestNotificationPermission: vi.fn(),
}));

import {
  getNotificationPermission,
  isNotificationSupported,
  requestNotificationPermission,
} from "@/lib/notifications";
import { useNotificationPermission } from "@/hooks/useNotificationPermission";

describe("useNotificationPermission", () => {
  beforeEach(() => {
    vi.mocked(isNotificationSupported).mockReturnValue(true);
    vi.mocked(getNotificationPermission).mockReturnValue("default");
    vi.mocked(requestNotificationPermission).mockResolvedValue("granted");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("reports unsupported browsers without reading permission", () => {
    vi.mocked(isNotificationSupported).mockReturnValue(false);
    vi.mocked(getNotificationPermission).mockReturnValue("denied");

    const { result } = renderHook(() => useNotificationPermission());

    expect(result.current.supported).toBe(false);
    expect(result.current.permission).toBe("denied");
  });

  it("exposes the current permission on a supported browser", () => {
    vi.mocked(getNotificationPermission).mockReturnValue("granted");

    const { result } = renderHook(() => useNotificationPermission());

    expect(result.current.supported).toBe(true);
    expect(result.current.permission).toBe("granted");
  });

  it("updates permission after request() resolves", async () => {
    const { result } = renderHook(() => useNotificationPermission());

    expect(result.current.permission).toBe("default");

    await act(async () => {
      await result.current.request();
    });

    expect(result.current.permission).toBe("granted");
    expect(requestNotificationPermission).toHaveBeenCalledTimes(1);
  });

  it("re-reads permission when the tab becomes visible again", () => {
    const { result } = renderHook(() => useNotificationPermission());
    expect(result.current.permission).toBe("default");

    vi.mocked(getNotificationPermission).mockReturnValue("denied");
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      configurable: true,
    });

    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(result.current.permission).toBe("denied");
  });
});
