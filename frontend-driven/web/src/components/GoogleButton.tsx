import { useRef, useEffect } from 'react';
import { initGoogleSdk, renderGoogleButton } from '../sdk/googleSdk';
import { GOOGLE_CLIENT_ID } from '../config/providers';
import { authenticate } from '../api/auth';
import { useAuth } from '../hooks/useAuth';

export function GoogleButton() {
  const containerRef = useRef<HTMLDivElement>(null);
  const { login } = useAuth();

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID || !containerRef.current) return;

    initGoogleSdk(GOOGLE_CLIENT_ID).then(() => {
      if (containerRef.current) {
        renderGoogleButton(containerRef.current, GOOGLE_CLIENT_ID, async (credential) => {
          try {
            const data = await authenticate('google', credential);
            login(data);
          } catch (err) {
            console.error('Google auth error:', err);
          }
        });
      }
    });
  }, [login]);

  if (!GOOGLE_CLIENT_ID) return null;

  return <div ref={containerRef} className="provider-slot" />;
}
