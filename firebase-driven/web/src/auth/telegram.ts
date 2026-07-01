import { signInWithCustomToken } from 'firebase/auth';
import { auth } from '../firebase';
import { TELEGRAM_FN_URL } from '../config/providers';
import { syncUser } from './syncUser';

export interface TelegramLoginStart {
  bot_username: string;
  deep_link: string;
  login_token: string;
}

export async function startTelegramLogin(): Promise<TelegramLoginStart> {
  const res = await fetch(`${TELEGRAM_FN_URL}/telegramStart`, {
    method: 'POST',
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Telegram login unavailable' }));
    throw new Error((err as { error: string }).error);
  }

  return res.json();
}

export async function verifyTelegramCode(loginToken: string, code: string): Promise<void> {
  const res = await fetch(`${TELEGRAM_FN_URL}/telegramVerify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login_token: loginToken, code }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Verification failed' }));
    throw new Error((err as { error: string }).error);
  }

  const { custom_token } = (await res.json()) as { custom_token: string };
  const result = await signInWithCustomToken(auth, custom_token);
  await syncUser(result.user);
}
