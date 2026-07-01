import { useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchMe, refreshToken, logout as apiLogout } from '../api/auth';
import type { User } from '../api/auth';

export function useAuth() {
  const queryClient = useQueryClient();

  const { data: user, isLoading, error } = useQuery<User>({
    queryKey: ['me'],
    queryFn: async () => {
      try {
        return await fetchMe();
      } catch {
        // Try refreshing token once
        try {
          await refreshToken();
          return await fetchMe();
        } catch {
          throw new Error('Not authenticated');
        }
      }
    },
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  const logout = async () => {
    await apiLogout();
    queryClient.setQueryData(['me'], null);
    queryClient.invalidateQueries({ queryKey: ['me'] });
  };

  return {
    user: user ?? null,
    isLoading,
    isAuthenticated: !!user && !error,
    logout,
  };
}
