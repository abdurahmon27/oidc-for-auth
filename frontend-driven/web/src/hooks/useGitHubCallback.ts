import { useEffect, useRef, useState } from 'react';
import { extractGitHubCallback } from '../providers/github';
import { authenticate } from '../api/auth';
import { useAuth } from './useAuth';

export function useGitHubCallback() {
  const { login } = useAuth();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const processedRef = useRef(false);

  useEffect(() => {
    if (processedRef.current) return;
    processedRef.current = true;

    async function handleCallback() {
      try {
        const token = extractGitHubCallback();
        if (!token) {
          setError('No authorization code found');
          return;
        }

        const data = await authenticate(token.provider, token.token);
        login(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'GitHub login failed');
      } finally {
        setIsLoading(false);
      }
    }

    handleCallback();
  }, [login]);

  return { isLoading, error };
}
