import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { clearIdToken, getIdToken, hasUsableSession, refreshAuthTokensIfNeeded, startTokenRefreshLoop } from "@/lib/auth";
import { parseIdToken } from "@/lib/idToken";
import { fetchAuthProviders, findProviderBySlug } from "@/lib/oidcProviders";
import { consumeSkipProfileOnCreate } from "@/lib/signInPreferences";
import { ensureUserRegistered } from "@/lib/users";

export type AuthUser = {
  id: string;
  subject: string;
  name?: string;
  provider: string;
  oidcProviderId: string;
  kemPublicKey: string;
};

export type OIDCProvider = {
  id: string;
  name: string;
  link: string;
  slug: string;
  picture?: string;
};

type AuthState = {
  user: AuthUser | null;
  providers: OIDCProvider[];
  loading: boolean;
  signOut: () => void;
  setDisplayName: (name?: string) => void;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [providers, setProviders] = useState<OIDCProvider[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    async function loadAuthState() {
      try {
        const providersResponse = await fetchAuthProviders().catch(() => null);
        if (!active) {
          return;
        }

        const loadedProviders = providersResponse ?? [];
        if (providersResponse) {
          setProviders(loadedProviders);
        }

        await refreshAuthTokensIfNeeded();

        if (!hasUsableSession()) {
          clearIdToken();
          setUser(null);
          return;
        }

        const token = getIdToken();
        if (!token) {
          setUser(null);
          return;
        }

        const tokenUser = parseIdToken(token);
        if (!tokenUser) {
          clearIdToken();
          setUser(null);
          return;
        }

        const provider = findProviderBySlug(loadedProviders, tokenUser.provider);
        if (!provider) {
          if (loadedProviders.length === 0) {
            setUser(null);
            return;
          }
          clearIdToken();
          setUser(null);
          return;
        }

        const skipProfile = consumeSkipProfileOnCreate();
        const dbUser = await ensureUserRegistered(tokenUser, provider.id, { skipProfile });
        if (!active) {
          return;
        }

        setUser({
          id: dbUser.id,
          subject: dbUser.subject,
          name: dbUser.name?.trim() || undefined,
          provider: tokenUser.provider,
          oidcProviderId: dbUser.oidcProviderId,
          kemPublicKey: dbUser.kemPublicKey,
        });
      } catch {
        if (active) {
          clearIdToken();
          setUser(null);
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void loadAuthState();

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!user) {
      return;
    }

    return startTokenRefreshLoop(() => {
      clearIdToken();
      setUser(null);
    });
  }, [user]);

  const signOut = useCallback(() => {
    clearIdToken();
    setUser(null);
  }, []);

  const setDisplayName = useCallback((name?: string) => {
    setUser((current) => (current ? { ...current, name } : null));
  }, []);

  const value = useMemo(
    () => ({ user, providers, loading, signOut, setDisplayName }),
    [user, providers, loading, signOut, setDisplayName],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
