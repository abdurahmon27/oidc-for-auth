import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { FACEBOOK_APP_ID } from '../config/providers';

export function FacebookButton() {
  const { loginWithProvider, isLoading } = useProviderLogin();

  if (!FACEBOOK_APP_ID) return null;

  return (
    <ProviderButton
      provider="facebook"
      label={isLoading ? 'Signing in…' : 'Continue with Facebook'}
      onClick={() => loginWithProvider('facebook')}
      disabled={isLoading}
    />
  );
}
