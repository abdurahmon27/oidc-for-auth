import type { ProviderToken } from '../types/auth';

export interface AuthProvider {
  name: string;
  authenticate(): Promise<ProviderToken>;
}
