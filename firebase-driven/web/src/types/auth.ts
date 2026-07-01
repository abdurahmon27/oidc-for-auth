import type { User as FirebaseUser } from 'firebase/auth';

export interface User {
  id: string;
  email?: string;
  phone?: string;
  name: string;
  avatar_url?: string;
  providers: ProviderInfo[];
}

export interface ProviderInfo {
  provider: string;
  email?: string;
  name?: string;
}

const PROVIDER_ID_MAP: Record<string, string> = {
  'google.com': 'google',
  'microsoft.com': 'microsoft',
  'facebook.com': 'facebook',
  'github.com': 'github',
  'custom': 'telegram',
};

export function normalizeProviderId(providerId: string): string {
  return PROVIDER_ID_MAP[providerId] ?? providerId;
}

export function mapFirebaseUser(fbUser: FirebaseUser): User {
  const providers: ProviderInfo[] = fbUser.providerData.map((p) => ({
    provider: normalizeProviderId(p.providerId),
    email: p.email ?? undefined,
    name: p.displayName ?? undefined,
  }));

  return {
    id: fbUser.uid,
    email: fbUser.email ?? undefined,
    phone: fbUser.phoneNumber ?? undefined,
    name: fbUser.displayName || fbUser.email || 'User',
    avatar_url: fbUser.photoURL ?? undefined,
    providers,
  };
}
