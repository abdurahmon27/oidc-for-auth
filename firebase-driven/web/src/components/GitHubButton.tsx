import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { GITHUB_ENABLED } from '../config/providers';

export function GitHubButton() {
  const { loginWithProvider, pendingProvider } = useProviderLogin();

  if (!GITHUB_ENABLED) return null;

  return (
    <ProviderButton
      provider="github"
      label={pendingProvider === 'github' ? 'Signing in…' : 'Continue with GitHub'}
      onClick={() => loginWithProvider('github')}
      disabled={pendingProvider !== null}
    />
  );
}
