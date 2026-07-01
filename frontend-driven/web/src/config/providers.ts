export const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID as string ?? '';
export const MICROSOFT_CLIENT_ID = import.meta.env.VITE_MICROSOFT_CLIENT_ID as string ?? '';
export const MICROSOFT_TENANT = (import.meta.env.VITE_MICROSOFT_TENANT as string) || 'common';
export const FACEBOOK_APP_ID = import.meta.env.VITE_FACEBOOK_APP_ID as string ?? '';
export const GITHUB_CLIENT_ID = import.meta.env.VITE_GITHUB_CLIENT_ID as string ?? '';
export const GITHUB_REDIRECT_URI = (import.meta.env.VITE_GITHUB_REDIRECT_URI as string) || `${window.location.origin}/auth/github/callback`;
