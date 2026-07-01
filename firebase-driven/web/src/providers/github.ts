import { GithubAuthProvider } from 'firebase/auth';
import type { AuthProvider } from './types';

export class GitHubProvider implements AuthProvider {
  name = 'github';

  build(): GithubAuthProvider {
    const provider = new GithubAuthProvider();
    provider.addScope('read:user');
    provider.addScope('user:email');
    return provider;
  }
}
