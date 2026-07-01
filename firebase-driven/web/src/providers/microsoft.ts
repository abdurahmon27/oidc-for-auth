import { OAuthProvider } from 'firebase/auth';
import type { AuthProvider } from './types';
import { MICROSOFT_TENANT } from '../config/providers';

export class MicrosoftProvider implements AuthProvider {
  name = 'microsoft';

  build(): OAuthProvider {
    const provider = new OAuthProvider('microsoft.com');
    provider.addScope('openid');
    provider.addScope('profile');
    provider.addScope('email');
    provider.setCustomParameters({ tenant: MICROSOFT_TENANT });
    return provider;
  }
}
