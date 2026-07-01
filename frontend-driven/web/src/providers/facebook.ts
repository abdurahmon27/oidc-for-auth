import type { AuthProvider } from './types';
import type { ProviderToken } from '../types/auth';
import { initFacebookSdk } from '../sdk/facebookSdk';
import { FACEBOOK_APP_ID } from '../config/providers';

export class FacebookProvider implements AuthProvider {
  name = 'facebook';

  async authenticate(): Promise<ProviderToken> {
    await initFacebookSdk(FACEBOOK_APP_ID);

    return new Promise((resolve, reject) => {
      if (!window.FB) {
        reject(new Error('Facebook SDK not loaded'));
        return;
      }

      window.FB.login(
        (response) => {
          if (response.authResponse?.accessToken) {
            resolve({
              provider: 'facebook',
              token: response.authResponse.accessToken,
              tokenType: 'access_token',
            });
          } else {
            reject(new Error('Facebook login cancelled or failed'));
          }
        },
        { scope: 'email,public_profile' }
      );
    });
  }
}
