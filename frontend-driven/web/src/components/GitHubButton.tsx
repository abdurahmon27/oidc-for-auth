import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { GITHUB_CLIENT_ID } from '../config/providers';

export function GitHubButton() {
  const { loginWithProvider, isLoading } = useProviderLogin();

  if (!GITHUB_CLIENT_ID) return null;

  return (
    <ProviderButton
      provider="github"
      label={isLoading ? 'Redirecting…' : 'Continue with GitHub'}
      onClick={() => loginWithProvider('github')}
      disabled={isLoading}
    />
  );
}
