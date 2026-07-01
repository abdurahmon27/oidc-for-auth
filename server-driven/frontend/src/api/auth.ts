const API_BASE = '/api';

export interface User {
  id: string;
  email?: string;
  phone?: string;
  name: string;
  avatar_url?: string;
  providers: { provider: string; email?: string; name?: string }[];
}

export async function fetchMe(): Promise<User> {
  const res = await fetch(`${API_BASE}/me`, { credentials: 'include' });
  if (!res.ok) {
    throw new Error('Not authenticated');
  }
  return res.json();
}

export async function refreshToken(): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!res.ok) {
    throw new Error('Refresh failed');
  }
}

export async function logout(): Promise<void> {
  const csrfToken = getCookie('csrf_token');
  await fetch(`${API_BASE}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'X-CSRF-Token': csrfToken || '',
    },
  });
}

export interface TelegramLoginStart {
  bot_username: string;
  deep_link: string;
  login_token: string;
}

export async function startTelegramLogin(): Promise<TelegramLoginStart> {
  const res = await fetch(`${API_BASE}/auth/telegram/start`, {
    method: 'POST',
    credentials: 'include',
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error || 'Telegram login unavailable');
  }
  return res.json();
}

export async function verifyTelegramCode(loginToken: string, code: string): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/telegram/verify`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login_token: loginToken, code }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error || 'Verification failed');
  }
}

export function getOAuthLoginURL(provider: string): string {
  return `${API_BASE}/auth/${provider}/login`;
}

function getCookie(name: string): string | null {
  const match = document.cookie.match(new RegExp(`(^| )${name}=([^;]+)`));
  return match ? match[2] : null;
}
