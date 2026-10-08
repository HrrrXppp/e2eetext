import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/notifications", () => ({
  isNotificationSupported: vi.fn(() => true),
  getNotificationPermission: vi.fn(() => "default"),
  requestNotificationPermission: vi.fn(async () => "granted"),
}));

import { useNotificationPermission } from "@/hooks/useNotificationPermission";
import { requestNotificationPermission } from "@/lib/notifications";

describe("useNotificationPermission", () => {
  beforeEach(() => {
    vi.mocked(requestNotificationPermission).mockClear();
  });

  it("starts with the current permission and updates after request", async () => {
    const { result } = renderHook(() => useNotificationPermission());
    expect(result.current.supported).toBe(true);
    expect(result.current.permission).toBe("default");

    await act(async () => {
      await result.current.request();
    });

    expect(requestNotificationPermission).toHaveBeenCalledTimes(1);
    expect(result.current.permission).toBe("granted");
  });
});
