import { useAuth } from '../hooks/useAuth';
import { OAuthButton } from '../components/OAuthButton';
import { TelegramForm } from '../components/TelegramForm';
import { LockIcon } from '../components/icons';
import { useNavigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export function LoginPage() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  if (isLoading) {
    return (
      <div className="loading">
        <div className="loading__inner">
          <div className="spinner" />
          <span>Loading…</span>
        </div>
      </div>
    );
  }

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

        <div className="provider-list">
          <OAuthButton provider="google" label="Continue with Google" />
          <OAuthButton provider="microsoft" label="Continue with Microsoft" />
          <OAuthButton provider="facebook" label="Continue with Facebook" />
          <OAuthButton provider="github" label="Continue with GitHub" />
        </div>

        <div className="divider">or</div>

        <TelegramForm
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['me'] });
            navigate('/dashboard');
          }}
        />
      </div>
    </div>
  );
}
