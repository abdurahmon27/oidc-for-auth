import { useAuthContext } from '../auth/AuthContext';

export function useProviderLogin() {
  const { loginWithProvider, pendingProvider, error } = useAuthContext();
  return { loginWithProvider, pendingProvider, isLoading: pendingProvider !== null, error };
}
