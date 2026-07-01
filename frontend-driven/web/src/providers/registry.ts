import type { AuthProvider } from './types';
import { GoogleProvider } from './google';
import { MicrosoftProvider } from './microsoft';
import { FacebookProvider } from './facebook';
import { GitHubProvider } from './github';
import {
  GOOGLE_CLIENT_ID,
  MICROSOFT_CLIENT_ID,
  FACEBOOK_APP_ID,
  GITHUB_CLIENT_ID,
} from '../config/providers';

const providers = new Map<string, AuthProvider>();

if (GOOGLE_CLIENT_ID) providers.set('google', new GoogleProvider());
if (MICROSOFT_CLIENT_ID) providers.set('microsoft', new MicrosoftProvider());
if (FACEBOOK_APP_ID) providers.set('facebook', new FacebookProvider());
if (GITHUB_CLIENT_ID) providers.set('github', new GitHubProvider());

export function getProvider(name: string): AuthProvider | undefined {
  return providers.get(name);
}

export function getAvailableProviders(): string[] {
  return Array.from(providers.keys());
}
