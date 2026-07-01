import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { GOOGLE_ENABLED } from '../config/providers';

export function GoogleButton() {
  const { loginWithProvider, pendingProvider } = useProviderLogin();

  if (!GOOGLE_ENABLED) return null;

  return (
    <ProviderButton
      provider="google"
      label={pendingProvider === 'google' ? 'Signing in…' : 'Continue with Google'}
      onClick={() => loginWithProvider('google')}
      disabled={pendingProvider !== null}
    />
  );
}
