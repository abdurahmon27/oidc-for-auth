import type { AuthProvider } from './types';
import type { ProviderToken } from '../types/auth';
import { GITHUB_CLIENT_ID, GITHUB_REDIRECT_URI } from '../config/providers';

export class GitHubProvider implements AuthProvider {
  name = 'github';

  async authenticate(): Promise<ProviderToken> {
    // GitHub requires full page redirect (no JS SDK)
    const state = crypto.randomUUID();
    sessionStorage.setItem('github_oauth_state', state);

    const params = new URLSearchParams({
      client_id: GITHUB_CLIENT_ID,
      redirect_uri: GITHUB_REDIRECT_URI,
      scope: 'user:email',
      state,
    });

    window.location.href = `https://github.com/login/oauth/authorize?${params.toString()}`;

    // This promise never resolves because we redirect away
    return new Promise(() => {});
  }
}

export function extractGitHubCallback(): ProviderToken | null {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('code');
  const state = params.get('state');

  if (!code || !state) return null;

  const savedState = sessionStorage.getItem('github_oauth_state');
  sessionStorage.removeItem('github_oauth_state');

  if (state !== savedState) {
    throw new Error('Invalid state parameter');
  }

  return {
    provider: 'github',
    token: code,
    tokenType: 'code',
  };
}
