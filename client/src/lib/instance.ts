export const DEV_INSTANCE_BANNER_TEXT =
  "This is a developer instance! Anything can be wrong!";

export type InstanceEnvironment = "production" | "development";

export type InstanceConfig = {
  environment: InstanceEnvironment;
};

export function isDevelopmentInstance(config: InstanceConfig | null): boolean {
  return config?.environment === "development";
}

let configPromise: Promise<InstanceConfig | null> | null = null;

export function fetchInstanceConfig(): Promise<InstanceConfig | null> {
  if (!configPromise) {
    configPromise = fetch("/instance.json", { cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) {
          return null;
        }
        const data = (await response.json()) as Partial<InstanceConfig>;
        if (data.environment === "development" || data.environment === "production") {
          return { environment: data.environment };
        }
        return null;
      })
      .catch(() => null);
  }
  return configPromise;
}

export function resetInstanceConfigCache(): void {
  configPromise = null;
}
