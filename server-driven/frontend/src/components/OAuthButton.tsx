import type { CSSProperties } from 'react';
import { getOAuthLoginURL } from '../api/auth';
import { ProviderIcon, BRAND_COLORS } from './icons';

interface OAuthButtonProps {
  provider: string;
  label: string;
}

export function OAuthButton({ provider, label }: OAuthButtonProps) {
  const style = { '--brand': BRAND_COLORS[provider] } as CSSProperties;
  return (
    <a href={getOAuthLoginURL(provider)} className="provider-btn" style={style}>
      <span className="provider-btn__icon">
        <ProviderIcon provider={provider} />
      </span>
      <span className="provider-btn__label">{label}</span>
    </a>
  );
}
