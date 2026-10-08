import { useCallback, useEffect, useState } from "react";
import {
  getNotificationPermission,
  isNotificationSupported,
  requestNotificationPermission,
} from "@/lib/notifications";

export type UseNotificationPermissionResult = {
  supported: boolean;
  permission: NotificationPermission;
  request: () => Promise<NotificationPermission>;
};

export function useNotificationPermission(): UseNotificationPermissionResult {
  const supported = isNotificationSupported();
  const [permission, setPermission] = useState<NotificationPermission>(() =>
    getNotificationPermission(),
  );

  useEffect(() => {
    if (!supported) {
      return;
    }

    // Permission can also change via the browser's own site-settings UI
    // (e.g. the address-bar lock icon) outside of any in-app action, so
    // re-read it when the tab regains focus -- mirrors the
    // visibilitychange-based refresh pattern in lib/auth.ts.
    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        setPermission(getNotificationPermission());
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, [supported]);

  const request = useCallback(async () => {
    const result = await requestNotificationPermission();
    setPermission(result);
    return result;
  }, []);

  return { supported, permission, request };
}
