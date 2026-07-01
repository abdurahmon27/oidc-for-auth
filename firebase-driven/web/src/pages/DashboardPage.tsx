import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { AuthLoading } from '../components/AuthLoading';
import { ProviderIcon } from '../components/icons';

export function DashboardPage() {
  const { user, isLoading, isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate('/');
    }
  }, [isLoading, isAuthenticated, navigate]);

  if (isLoading) return <AuthLoading />;
  if (!user) return null;

  const handleLogout = async () => {
    await logout();
    navigate('/');
  };

  const initial = (user.name || user.email || 'U').charAt(0).toUpperCase();

  return (
    <div className="auth-page">
      <div className="auth-card auth-card--wide">
        <div className="auth-head">
          <h1 className="auth-title">Dashboard</h1>
        </div>

        <div className="profile">
          {user.avatar_url ? (
            <img src={user.avatar_url} alt="" className="profile__avatar" />
          ) : (
            <div className="profile__avatar profile__avatar--fallback">{initial}</div>
          )}
          <div>
            <p className="profile__name">{user.name || 'User'}</p>
            {user.email && <p className="profile__meta">{user.email}</p>}
            {user.phone && <p className="profile__meta">{user.phone}</p>}
            <p className="profile__meta">ID: {user.id}</p>
          </div>
        </div>

        <p className="section-label">Linked providers</p>
        {user.providers.length === 0 ? (
          <ul className="linked">
            <li className="linked--empty">No providers linked</li>
          </ul>
        ) : (
          <ul className="linked">
            {user.providers.map((p) => (
              <li key={p.provider} className="linked__row">
                <span className="linked__icon">
                  <ProviderIcon provider={p.provider} />
                </span>
                <span className="linked__name">{p.provider}</span>
                {p.email && <span className="linked__email">{p.email}</span>}
              </li>
            ))}
          </ul>
        )}

        <button onClick={handleLogout} className="btn btn--ghost">
          Log out
        </button>
      </div>
    </div>
  );
}
