import { useEffect, useState } from "react";
import { fetchInstanceConfig, isDevelopmentInstance } from "@/lib/instance";

export function useDevInstance(): boolean {
  const [show, setShow] = useState(false);

  useEffect(() => {
    let active = true;

    void fetchInstanceConfig().then((config) => {
      if (active) {
        setShow(isDevelopmentInstance(config));
      }
    });

    return () => {
      active = false;
    };
  }, []);

  return show;
}
