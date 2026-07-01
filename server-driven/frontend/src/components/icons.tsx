/* Brand + UI icons — shared identically across both web apps. */
import type { CSSProperties, ReactElement } from 'react';

type IconProps = { style?: CSSProperties };

export function GoogleIcon(_: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="#4285F4" d="M23.52 12.27c0-.82-.07-1.6-.2-2.36H12v4.46h6.47a5.53 5.53 0 0 1-2.4 3.63v3.02h3.88c2.27-2.09 3.57-5.17 3.57-8.75z" />
      <path fill="#34A853" d="M12 24c3.24 0 5.96-1.08 7.95-2.91l-3.88-3.02c-1.08.72-2.45 1.15-4.07 1.15-3.13 0-5.78-2.11-6.73-4.96H1.29v3.12A12 12 0 0 0 12 24z" />
      <path fill="#FBBC05" d="M5.27 14.26a7.2 7.2 0 0 1 0-4.52V6.62H1.29a12 12 0 0 0 0 10.76l3.98-3.12z" />
      <path fill="#EA4335" d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.44-3.44C17.95 1.19 15.24 0 12 0A12 12 0 0 0 1.29 6.62l3.98 3.12C6.22 6.86 8.87 4.75 12 4.75z" />
    </svg>
  );
}

export function MicrosoftIcon(_: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="#F25022" d="M1 1h10.5v10.5H1z" />
      <path fill="#7FBA00" d="M12.5 1H23v10.5H12.5z" />
      <path fill="#00A4EF" d="M1 12.5h10.5V23H1z" />
      <path fill="#FFB900" d="M12.5 12.5H23V23H12.5z" />
    </svg>
  );
}

export function FacebookIcon(_: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="#1877F2" d="M24 12a12 12 0 1 0-13.88 11.85v-8.38H7.08V12h3.04V9.36c0-3 1.79-4.67 4.53-4.67 1.31 0 2.68.24 2.68.24v2.95h-1.51c-1.49 0-1.96.93-1.96 1.87V12h3.33l-.53 3.47h-2.8v8.38A12 12 0 0 0 24 12z" />
    </svg>
  );
}

export function GitHubIcon(_: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="currentColor" d="M12 .5A11.5 11.5 0 0 0 .5 12a11.5 11.5 0 0 0 7.86 10.92c.58.1.79-.25.79-.56v-2c-3.2.7-3.88-1.37-3.88-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.2 1.77 1.2 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.79 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.84 1.19 3.1 0 4.42-2.69 5.4-5.25 5.68.41.36.78 1.05.78 2.12v3.15c0 .31.21.67.8.56A11.5 11.5 0 0 0 23.5 12 11.5 11.5 0 0 0 12 .5z" />
    </svg>
  );
}

export function TelegramIcon({ style }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" style={style}>
      <path fill="currentColor" d="M22.05 3.4 2.9 10.79c-1.3.52-1.3 1.26-.24 1.58l4.9 1.53 1.9 5.82c.23.63.11.88.77.88.51 0 .74-.23 1.02-.5l2.46-2.4 5.11 3.78c.94.52 1.62.25 1.85-.87l3.36-15.83c.34-1.37-.53-1.99-1.98-1.36z" />
    </svg>
  );
}

export function LockIcon(_: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <rect x="4.5" y="10.5" width="15" height="10" rx="2.2" stroke="currentColor" strokeWidth="1.8" />
      <path d="M8 10.5V7.5a4 4 0 0 1 8 0v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
    </svg>
  );
}

const BRAND_ICONS: Record<string, (p: IconProps) => ReactElement> = {
  google: GoogleIcon,
  microsoft: MicrosoftIcon,
  facebook: FacebookIcon,
  github: GitHubIcon,
  telegram: TelegramIcon,
};

export function ProviderIcon({ provider }: { provider: string }) {
  const Icon = BRAND_ICONS[provider.toLowerCase()];
  return Icon ? <Icon /> : null;
}

export const BRAND_COLORS: Record<string, string> = {
  google: '#4285F4',
  microsoft: '#00A4EF',
  facebook: '#1877F2',
  github: '#6e7681',
  telegram: '#229ED9',
};
