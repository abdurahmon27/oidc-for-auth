import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import {
  onAuthStateChanged,
  signOut,
  signInWithPopup,
  linkWithCredential,
  fetchSignInMethodsForEmail,
  GoogleAuthProvider,
  FacebookAuthProvider,
  GithubAuthProvider,
  OAuthProvider,
  type AuthError,
  type AuthCredential,
} from 'firebase/auth';
import { doc, getDoc } from 'firebase/firestore';
import { auth, db } from '../firebase';
import { getProvider } from '../providers/registry';
import { mapFirebaseUser, type User } from '../types/auth';
import { syncUser } from './syncUser';

function credentialFromError(providerName: string, error: AuthError): AuthCredential | null {
  switch (providerName) {
    case 'google':
      return GoogleAuthProvider.credentialFromError(error);
    case 'facebook':
      return FacebookAuthProvider.credentialFromError(error);
    case 'github':
      return GithubAuthProvider.credentialFromError(error);
    case 'microsoft':
      return OAuthProvider.credentialFromError(error);
    default:
      return null;
  }
}

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  pendingProvider: string | null;
  error: string | null;
  isAuthenticated: boolean;
  loginWithProvider: (name: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [pendingProvider, setPendingProvider] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Stashed when a popup sign-in collides with an existing account, so it can
  // be linked once the user signs in with their original provider.
  const pendingCredential = useRef<AuthCredential | null>(null);

  // Single auth-state subscription for the whole app. Reflect the session
  // synchronously — never block the authenticated state on Firestore.
  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, (fbUser) => {
      if (!fbUser) {
        setUser(null);
        setIsLoading(false);
        return;
      }

      setUser(mapFirebaseUser(fbUser));
      setIsLoading(false);

      // Mirror to Firestore and enrich providers (e.g. Telegram) in the background.
      void (async () => {
        await syncUser(fbUser);
        const mapped = mapFirebaseUser(fbUser);
        if (mapped.providers.length === 0) {
          try {
            const snap = await getDoc(doc(db, 'users', fbUser.uid));
            const providers = (snap.data()?.providers as string[] | undefined) ?? [];
            if (providers.length > 0) {
              setUser({ ...mapped, providers: providers.map((provider) => ({ provider })) });
            }
          } catch {
            // ignore — keep the Auth-derived user
          }
        }
      })();
    });

    return unsubscribe;
  }, []);

  const loginWithProvider = useCallback(async (name: string) => {
    setError(null);
    const provider = getProvider(name);
    if (!provider) {
      setError(`Provider ${name} not available`);
      return;
    }

    setPendingProvider(name);
    try {
      const result = await signInWithPopup(auth, provider.build());

      // Link a credential stashed from an earlier collision, if any.
      if (pendingCredential.current) {
        const cred = pendingCredential.current;
        pendingCredential.current = null;
        try {
          await linkWithCredential(result.user, cred);
        } catch {
          // ignore — user is still signed in with their original provider
        }
      }

      await syncUser(result.user);
      // onAuthStateChanged flips isAuthenticated → LoginPage redirects.
    } catch (err) {
      const authError = err as AuthError;

      if (authError.code === 'auth/account-exists-with-different-credential') {
        const email = authError.customData?.email as string | undefined;
        pendingCredential.current = credentialFromError(name, authError);
        let hint = 'your original provider';
        if (email) {
          try {
            const methods = await fetchSignInMethodsForEmail(auth, email);
            if (methods[0]) hint = methods[0];
          } catch {
            // ignore
          }
        }
        setError(
          `An account already exists${email ? ` for ${email}` : ''}. Sign in with ${hint} first — ` +
            `${name} will then link automatically.`
        );
      } else if (
        authError.code === 'auth/popup-closed-by-user' ||
        authError.code === 'auth/cancelled-popup-request'
      ) {
        // User dismissed the popup — not an error worth surfacing.
      } else if (authError?.message) {
        setError(authError.message);
      }
    } finally {
      setPendingProvider(null);
    }
  }, []);

  const logout = useCallback(async () => {
    await signOut(auth);
    setUser(null);
  }, []);

  const value: AuthContextValue = {
    user,
    isLoading,
    pendingProvider,
    error,
    isAuthenticated: !!user,
    loginWithProvider,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuthContext(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuthContext must be used within AuthProvider');
  return ctx;
}
