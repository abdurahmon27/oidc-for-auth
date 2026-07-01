import type { AuthProvider as FirebaseAuthProvider } from 'firebase/auth';

export interface AuthProvider {
  name: string;
  build(): FirebaseAuthProvider;
}
