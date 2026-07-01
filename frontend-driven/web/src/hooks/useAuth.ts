import { useCallback, useEffect, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { refresh, logout as apiLogout, getAccessToken, setAccessToken } from '../api/auth';
import type { User, AuthResponse } from '../types/auth';

export function useAuth() {
  const queryClient = useQueryClient();
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout>>(null);

  const scheduleRefresh = useCallback(
    (expiresIn: number) => {
      if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
      // Refresh 60 seconds before expiry
      const delay = Math.max((expiresIn - 60) * 1000, 5000);
      refreshTimerRef.current = setTimeout(async () => {
        try {
          const data = await refresh();
          queryClient.setQueryData<User>(['me'], data.user);
          scheduleRefresh(data.expires_in);
        } catch {
          setAccessToken(null);
          queryClient.setQueryData(['me'], null);
        }
      }, delay);
    },
    [queryClient]
  );

  const { data: user, isLoading } = useQuery<User | null>({
    queryKey: ['me'],
    queryFn: async () => {
      // On mount, try silent refresh to restore session
      if (!getAccessToken()) {
        try {
          const data = await refresh();
          scheduleRefresh(data.expires_in);
          return data.user;
        } catch {
          return null;
        }
      }
      return null;
    },
    retry: false,
    staleTime: Infinity,
  });

  useEffect(() => {
    return () => {
      if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    };
  }, []);

  const login = useCallback(
    (data: AuthResponse) => {
      queryClient.setQueryData<User>(['me'], data.user);
      scheduleRefresh(data.expires_in);
    },
    [queryClient, scheduleRefresh]
  );

  const logout = useCallback(async () => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    await apiLogout();
    queryClient.setQueryData(['me'], null);
  }, [queryClient]);

  return {
    user: user ?? null,
    isLoading,
    isAuthenticated: !!user,
    login,
    logout,
  };
}
