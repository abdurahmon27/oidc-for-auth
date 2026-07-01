import type { CSSProperties } from 'react';
import { ProviderIcon, BRAND_COLORS } from './icons';

interface ProviderButtonProps {
  provider: string;
  label: string;
  onClick: () => void;
  disabled?: boolean;
}

export function ProviderButton({ provider, label, onClick, disabled }: ProviderButtonProps) {
  const style = { '--brand': BRAND_COLORS[provider] } as CSSProperties;
  return (
    <button onClick={onClick} disabled={disabled} className="provider-btn" style={style}>
      <span className="provider-btn__icon">
        <ProviderIcon provider={provider} />
      </span>
      <span className="provider-btn__label">{label}</span>
    </button>
  );
}
