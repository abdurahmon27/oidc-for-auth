import type { AuthResponse, User } from '../types/auth';

const API_BASE = '/api';

let accessToken: string | null = null;

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export async function authenticate(
  provider: string,
  token: string
): Promise<AuthResponse> {
  const body: Record<string, unknown> = { provider, token };

  const res = await fetch(`${API_BASE}/auth/token`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Authentication failed' }));
    throw new Error((err as { error: string }).error);
  }

  const data: AuthResponse = await res.json();
  accessToken = data.access_token;
  return data;
}

export async function refresh(): Promise<AuthResponse> {
  const res = await fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    credentials: 'include',
  });

  if (!res.ok) {
    accessToken = null;
    throw new Error('Refresh failed');
  }

  const data: AuthResponse = await res.json();
  accessToken = data.access_token;
  return data;
}

export async function logout(): Promise<void> {
  await fetch(`${API_BASE}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  });
  accessToken = null;
}

export async function fetchMe(): Promise<User> {
  if (!accessToken) throw new Error('No access token');

  const res = await fetch(`${API_BASE}/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });

  if (!res.ok) throw new Error('Not authenticated');
  return res.json();
}

export interface TelegramLoginStart {
  bot_username: string;
  deep_link: string;
  login_token: string;
}

export async function startTelegramLogin(): Promise<TelegramLoginStart> {
  const res = await fetch(`${API_BASE}/auth/telegram/start`, {
    method: 'POST',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Telegram login unavailable' }));
    throw new Error((err as { error: string }).error);
  }

  return res.json();
}

export async function verifyTelegramCode(
  loginToken: string,
  code: string
): Promise<AuthResponse> {
  const res = await fetch(`${API_BASE}/auth/telegram/verify`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login_token: loginToken, code }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Verification failed' }));
    throw new Error((err as { error: string }).error);
  }

  const data: AuthResponse = await res.json();
  accessToken = data.access_token;
  return data;
}
