export const GOOGLE_ENABLED = (import.meta.env.VITE_GOOGLE_ENABLED as string) === 'true';
export const MICROSOFT_ENABLED = (import.meta.env.VITE_MICROSOFT_ENABLED as string) === 'true';
export const MICROSOFT_TENANT = (import.meta.env.VITE_MICROSOFT_TENANT as string) || 'common';
export const FACEBOOK_ENABLED = (import.meta.env.VITE_FACEBOOK_ENABLED as string) === 'true';
export const GITHUB_ENABLED = (import.meta.env.VITE_GITHUB_ENABLED as string) === 'true';
export const TELEGRAM_ENABLED = (import.meta.env.VITE_TELEGRAM_ENABLED as string) === 'true';
export const TELEGRAM_FN_URL = (import.meta.env.VITE_TELEGRAM_FN_URL as string) ?? '';
