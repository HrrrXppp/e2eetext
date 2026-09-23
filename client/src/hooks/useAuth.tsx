import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { OAUTH_CALLBACK_PATH } from "@/lib/api";
import {
  clearIdToken,
  getAuthProvider,
  getIdToken,
  hasUsableSession,
  refreshAuthTokensIfNeeded,
  startTokenRefreshLoop,
} from "@/lib/auth";
import { parseIdToken } from "@/lib/idToken";
import { ensureIdentityKeys } from "@/lib/e2ee/session";
import { clearE2EEState } from "@/lib/e2ee/storage";
import { fetchAuthProviders, findProviderBySlug } from "@/lib/oidcProviders";
import { consumePendingSignInName, consumeSkipProfileOnCreate } from "@/lib/signInPreferences";
import { ensureUserRegistered } from "@/lib/users";

export type AuthUser = {
  id: string;
  subject: string;
  name?: string;
  provider: string;
  oidcProviderId: string;
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
  // True for the remainder of this session only, starting the moment
  // ensureIdentityKeys generates a brand-new local keypair (new account or
  // new device). Never persisted, so it does not reappear on later sign-ins.
  justCreatedIdentity: boolean;
  acknowledgeIdentityBackup: () => void;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [providers, setProviders] = useState<OIDCProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [justCreatedIdentity, setJustCreatedIdentity] = useState(false);

  useEffect(() => {
    // AuthCallback.tsx owns this transient route: it stores the session
    // then immediately calls window.location.replace("/chats"), where a
    // fresh AuthProvider mount does the real work below. Nothing on this
    // route reads AuthContext (App.tsx renders only <AuthCallback />
    // here), and registering here too would race that navigation — the
    // in-flight fetch gets aborted mid-navigation, which can otherwise
    // surface as a spurious sign-out immediately after a successful login.
    if (window.location.pathname === OAUTH_CALLBACK_PATH) {
      setLoading(false);
      return;
    }

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

        // The provider slug is set explicitly at sign-in time (from the
        // server's OAuth callback redirect — see AuthCallback.tsx) and
        // persisted across reloads. Falling back to a guess derived from
        // the ID token's issuer only covers stored sessions predating that
        // change; it cannot reliably distinguish between multiple
        // non-Google providers (e.g. two different mock/OIDC providers, or
        // Apple vs. a generic OIDC provider).
        const providerSlug = getAuthProvider() ?? tokenUser.provider;
        const provider = findProviderBySlug(loadedProviders, providerSlug);
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
        const pendingName = consumePendingSignInName();
        const dbUser = await ensureUserRegistered(tokenUser, provider.id, {
          skipProfile,
          name: pendingName,
        });
        if (!active) {
          return;
        }

        // Awaited (not fire-and-forget) so the UI never becomes interactive
        // before a *local* identity keypair exists — chat creation and
        // message send both assume one is already in place. The server
        // upload inside ensureIdentityKeys is background work and doesn't
        // hold this up (see its comment) — this only throws on a genuine
        // local-storage/crypto failure. On failure this still falls through
        // to setUser below; chat creation/sending retries key generation on
        // demand.
        try {
          const identityResult = await ensureIdentityKeys(dbUser.id);
          if (active && identityResult.created) {
            setJustCreatedIdentity(true);
          }
        } catch {
          // Local generation/persistence failed; chat creation/sending
          // retries key generation on demand, so sign-in can still proceed.
        }
        if (!active) {
          return;
        }

        setUser({
          id: dbUser.id,
          subject: dbUser.subject,
          name: dbUser.name?.trim() || undefined,
          provider: tokenUser.provider,
          oidcProviderId: dbUser.oidcProviderId,
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
      if (user?.id) {
        void clearE2EEState(user.id);
      }
      clearIdToken();
      setUser(null);
    });
  }, [user]);

  const signOut = useCallback(() => {
    if (user?.id) {
      void clearE2EEState(user.id);
    }
    clearIdToken();
    setUser(null);
  }, [user?.id]);

  const setDisplayName = useCallback((name?: string) => {
    setUser((current) => (current ? { ...current, name } : null));
  }, []);

  const acknowledgeIdentityBackup = useCallback(() => {
    setJustCreatedIdentity(false);
  }, []);

  const value = useMemo(
    () => ({
      user,
      providers,
      loading,
      signOut,
      setDisplayName,
      justCreatedIdentity,
      acknowledgeIdentityBackup,
    }),
    [
      user,
      providers,
      loading,
      signOut,
      setDisplayName,
      justCreatedIdentity,
      acknowledgeIdentityBackup,
    ],
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
