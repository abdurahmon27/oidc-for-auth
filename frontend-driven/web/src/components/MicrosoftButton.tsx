import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { MICROSOFT_CLIENT_ID } from '../config/providers';

export function MicrosoftButton() {
  const { loginWithProvider, isLoading } = useProviderLogin();

  if (!MICROSOFT_CLIENT_ID) return null;

  return (
    <ProviderButton
      provider="microsoft"
      label={isLoading ? 'Signing in…' : 'Continue with Microsoft'}
      onClick={() => loginWithProvider('microsoft')}
      disabled={isLoading}
    />
  );
}
