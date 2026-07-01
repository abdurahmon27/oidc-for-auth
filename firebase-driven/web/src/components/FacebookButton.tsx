import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { FACEBOOK_ENABLED } from '../config/providers';

export function FacebookButton() {
  const { loginWithProvider, pendingProvider } = useProviderLogin();

  if (!FACEBOOK_ENABLED) return null;

  return (
    <ProviderButton
      provider="facebook"
      label={pendingProvider === 'facebook' ? 'Signing in…' : 'Continue with Facebook'}
      onClick={() => loginWithProvider('facebook')}
      disabled={pendingProvider !== null}
    />
  );
}
