import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useProviderLogin } from '../hooks/useProviderLogin';
import { GoogleButton } from '../components/GoogleButton';
import { MicrosoftButton } from '../components/MicrosoftButton';
import { FacebookButton } from '../components/FacebookButton';
import { GitHubButton } from '../components/GitHubButton';
import { TelegramForm } from '../components/TelegramForm';
import { AuthLoading } from '../components/AuthLoading';
import { AuthError } from '../components/AuthError';
import { LockIcon } from '../components/icons';
import { TELEGRAM_ENABLED } from '../config/providers';

export function LoginPage() {
  const { isAuthenticated, isLoading } = useAuth();
  const { error } = useProviderLogin();
  const navigate = useNavigate();

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  if (isLoading) return <AuthLoading />;

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-head">
          <div className="auth-mark">
            <LockIcon />
          </div>
          <h1 className="auth-title">Sign in</h1>
          <p className="auth-subtitle">Continue with your preferred account</p>
        </div>

        {error && <AuthError message={error} />}

        <div className="provider-list">
          <GoogleButton />
          <MicrosoftButton />
          <FacebookButton />
          <GitHubButton />
        </div>

        {TELEGRAM_ENABLED && (
          <>
            <div className="divider">or</div>
            <TelegramForm />
          </>
        )}
      </div>
    </div>
  );
}
