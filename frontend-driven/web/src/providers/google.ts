import type { AuthProvider } from './types';
import type { ProviderToken } from '../types/auth';
import { initGoogleSdk } from '../sdk/googleSdk';
import { GOOGLE_CLIENT_ID } from '../config/providers';

export class GoogleProvider implements AuthProvider {
  name = 'google';

  async authenticate(): Promise<ProviderToken> {
    await initGoogleSdk(GOOGLE_CLIENT_ID);

    return new Promise((resolve, reject) => {
      if (!window.google) {
        reject(new Error('Google SDK not loaded'));
        return;
      }

      window.google.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: (response) => {
          resolve({
            provider: 'google',
            token: response.credential,
            tokenType: 'id_token',
          });
        },
      });

      window.google.accounts.id.prompt();
    });
  }
}
