import { useCallback, useState } from "react";
import {
  getNotificationPermission,
  isNotificationSupported,
  requestNotificationPermission,
  type NotificationPermissionState,
} from "@/lib/notifications";

export function useNotificationPermission() {
  const [permission, setPermission] = useState<NotificationPermissionState>(() =>
    getNotificationPermission(),
  );

  const request = useCallback(async () => {
    setPermission(await requestNotificationPermission());
  }, []);

  return { supported: isNotificationSupported(), permission, request };
}
