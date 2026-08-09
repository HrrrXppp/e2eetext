import type { OIDCProvider } from "@/hooks/useAuth";
import { API_V1 } from "@/lib/api";

let providersRequest: Promise<OIDCProvider[]> | null = null;

export async function fetchAuthProviders(): Promise<OIDCProvider[]> {
  if (!providersRequest) {
    providersRequest = loadAuthProviders().catch((error) => {
      providersRequest = null;
      throw error;
    });
  }

  return providersRequest;
}

async function loadAuthProviders(): Promise<OIDCProvider[]> {
  const response = await fetch(`${API_V1}/auth/providers`);
  if (!response.ok) {
    throw new Error("failed to load auth providers");
  }

  return response.json() as Promise<OIDCProvider[]>;
}

export function resetAuthProvidersCache(): void {
  providersRequest = null;
}

export function findProviderBySlug(
  providers: OIDCProvider[],
  slug: string,
): OIDCProvider | undefined {
  return providers.find((provider) => provider.slug === slug);
}
