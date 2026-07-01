import { useAuthContext } from '../auth/AuthContext';

export function useAuth() {
  const { user, isLoading, isAuthenticated, logout } = useAuthContext();
  return { user, isLoading, isAuthenticated, logout };
}
