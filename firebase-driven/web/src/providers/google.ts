import { GoogleAuthProvider } from 'firebase/auth';
import type { AuthProvider } from './types';

export class GoogleProvider implements AuthProvider {
  name = 'google';

  build(): GoogleAuthProvider {
    return new GoogleAuthProvider();
  }
}
