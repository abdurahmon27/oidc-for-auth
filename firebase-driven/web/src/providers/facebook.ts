import { FacebookAuthProvider } from 'firebase/auth';
import type { AuthProvider } from './types';

export class FacebookProvider implements AuthProvider {
  name = 'facebook';

  build(): FacebookAuthProvider {
    return new FacebookAuthProvider();
  }
}
