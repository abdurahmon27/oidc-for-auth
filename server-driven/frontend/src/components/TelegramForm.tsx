import { useState } from 'react';
import { startTelegramLogin, verifyTelegramCode, type TelegramLoginStart } from '../api/auth';
import { TelegramIcon } from './icons';

interface TelegramFormProps {
  onSuccess: () => void;
}

type Step = 'idle' | 'started' | 'code';

export function TelegramForm({ onSuccess }: TelegramFormProps) {
  const [step, setStep] = useState<Step>('idle');
  const [session, setSession] = useState<TelegramLoginStart | null>(null);
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleStart = async () => {
    setError('');
    setLoading(true);
    try {
      const data = await startTelegramLogin();
      setSession(data);
      setStep('started');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Telegram login unavailable');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyCode = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!session) return;
    setError('');
    setLoading(true);
    try {
      await verifyTelegramCode(session.login_token, code);
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setLoading(false);
    }
  };

  const reset = () => {
    setStep('idle');
    setSession(null);
    setCode('');
    setError('');
  };

  return (
    <div className="tg">
      <p className="tg__head">
        <TelegramIcon />
        Telegram
      </p>
      {error && <div className="alert">{error}</div>}

      {step === 'idle' && (
        <button type="button" onClick={handleStart} disabled={loading} className="btn btn--tg">
          {loading ? 'Starting…' : 'Continue with Telegram'}
        </button>
      )}

      {step === 'started' && session && (
        <div className="stack">
          <p className="tg__step">
            1. Open <strong>@{session.bot_username}</strong> and press <strong>Start</strong>.
          </p>
          <a
            href={session.deep_link}
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn--tg"
          >
            Open @{session.bot_username}
          </a>
          <p className="tg__step">2. The bot sends a 6-digit code — enter it next.</p>
          <button type="button" onClick={() => setStep('code')} className="btn btn--primary">
            Enter code
          </button>
          <button type="button" onClick={reset} className="btn btn--ghost">
            Cancel
          </button>
        </div>
      )}

      {step === 'code' && session && (
        <form onSubmit={handleVerifyCode} className="stack">
          <p className="tg__step muted">Enter the code @{session.bot_username} sent you:</p>
          <input
            type="text"
            inputMode="numeric"
            placeholder="000000"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            maxLength={6}
            className="field field--otp"
            required
            autoFocus
          />
          <button type="submit" disabled={loading} className="btn btn--tg">
            {loading ? 'Verifying…' : 'Verify'}
          </button>
          <button type="button" onClick={() => setStep('started')} className="btn btn--ghost">
            Back
          </button>
        </form>
      )}
    </div>
  );
}
