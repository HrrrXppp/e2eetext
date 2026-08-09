import { useEffect } from "react";
import { useDevInstance } from "@/hooks/useDevInstance";
import { DEV_INSTANCE_BANNER_TEXT } from "@/lib/instance";

export function DevInstanceBanner() {
  const show = useDevInstance();

  useEffect(() => {
    document.body.classList.toggle("has-dev-instance-banner", show);
    return () => {
      document.body.classList.remove("has-dev-instance-banner");
    };
  }, [show]);

  if (!show) {
    return null;
  }

  return (
    <div className="dev-instance-banner" role="status">
      {DEV_INSTANCE_BANNER_TEXT}
    </div>
  );
}
