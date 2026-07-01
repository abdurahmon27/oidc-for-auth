import type { AuthProvider } from './types';
import type { ProviderToken } from '../types/auth';
import { getMsalInstance } from '../sdk/microsoftSdk';
import { MICROSOFT_CLIENT_ID, MICROSOFT_TENANT } from '../config/providers';

export class MicrosoftProvider implements AuthProvider {
  name = 'microsoft';

  async authenticate(): Promise<ProviderToken> {
    const msal = await getMsalInstance(MICROSOFT_CLIENT_ID, MICROSOFT_TENANT);

    const result = await msal.loginPopup({
      scopes: ['openid', 'profile', 'email'],
    });

    if (!result.idToken) {
      throw new Error('Microsoft login did not return an id_token');
    }

    return {
      provider: 'microsoft',
      token: result.idToken,
      tokenType: 'id_token',
    };
  }
}
