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

export interface AuthResponse {
  access_token: string;
  expires_in: number;
  user: User;
}

export interface ProviderToken {
  provider: string;
  token: string;
  tokenType: 'id_token' | 'access_token' | 'code';
}
