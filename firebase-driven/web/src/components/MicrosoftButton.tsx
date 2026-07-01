import { useProviderLogin } from '../hooks/useProviderLogin';
import { ProviderButton } from './ProviderButton';
import { MICROSOFT_ENABLED } from '../config/providers';

export function MicrosoftButton() {
  const { loginWithProvider, pendingProvider } = useProviderLogin();

  if (!MICROSOFT_ENABLED) return null;

  return (
    <ProviderButton
      provider="microsoft"
      label={pendingProvider === 'microsoft' ? 'Signing in…' : 'Continue with Microsoft'}
      onClick={() => loginWithProvider('microsoft')}
      disabled={pendingProvider !== null}
    />
  );
}
