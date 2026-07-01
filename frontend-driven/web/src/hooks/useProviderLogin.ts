import { useState, useCallback } from 'react';
import { getProvider } from '../providers/registry';
import { authenticate } from '../api/auth';
import { useAuth } from './useAuth';
import type { AuthResponse } from '../types/auth';

export function useProviderLogin() {
  const { login } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loginWithProvider = useCallback(
    async (providerName: string) => {
      setError(null);
      setIsLoading(true);

      try {
        const provider = getProvider(providerName);
        if (!provider) throw new Error(`Provider ${providerName} not available`);

        const providerToken = await provider.authenticate();

        const data: AuthResponse = await authenticate(
          providerToken.provider,
          providerToken.token
        );

        login(data);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Login failed';
        setError(message);
      } finally {
        setIsLoading(false);
      }
    },
    [login]
  );

  return { loginWithProvider, isLoading, error };
}
