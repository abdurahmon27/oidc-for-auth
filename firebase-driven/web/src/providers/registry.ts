import type { AuthProvider } from './types';
import { GoogleProvider } from './google';
import { MicrosoftProvider } from './microsoft';
import { FacebookProvider } from './facebook';
import { GitHubProvider } from './github';
import {
  GOOGLE_ENABLED,
  MICROSOFT_ENABLED,
  FACEBOOK_ENABLED,
  GITHUB_ENABLED,
} from '../config/providers';

const providers = new Map<string, AuthProvider>();

if (GOOGLE_ENABLED) providers.set('google', new GoogleProvider());
if (MICROSOFT_ENABLED) providers.set('microsoft', new MicrosoftProvider());
if (FACEBOOK_ENABLED) providers.set('facebook', new FacebookProvider());
if (GITHUB_ENABLED) providers.set('github', new GitHubProvider());

export function getProvider(name: string): AuthProvider | undefined {
  return providers.get(name);
}

export function getAvailableProviders(): string[] {
  return Array.from(providers.keys());
}
