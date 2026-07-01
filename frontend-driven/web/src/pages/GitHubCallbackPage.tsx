import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useGitHubCallback } from '../hooks/useGitHubCallback';
import { useAuth } from '../hooks/useAuth';
import { AuthLoading } from '../components/AuthLoading';
import { AuthError } from '../components/AuthError';

export function GitHubCallbackPage() {
  const { isLoading, error } = useGitHubCallback();
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  if (isLoading) return <AuthLoading />;

  if (error) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <AuthError message={error} />
          <button onClick={() => navigate('/')} className="btn btn--ghost">
            Back to login
          </button>
        </div>
      </div>
    );
  }

  return <AuthLoading />;
}
