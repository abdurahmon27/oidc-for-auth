import { loadScript } from './loadScript';

let initialized = false;

export async function initGoogleSdk(clientId: string): Promise<void> {
  if (initialized) return;
  await loadScript('https://accounts.google.com/gsi/client');
  if (!window.google) throw new Error('Google SDK failed to load');
  initialized = true;
  // Initialization happens per-button via google.accounts.id.initialize
  void clientId; // used by caller when calling initialize
}

export function renderGoogleButton(
  element: HTMLElement,
  clientId: string,
  callback: (credential: string) => void
): void {
  if (!window.google) return;

  window.google.accounts.id.initialize({
    client_id: clientId,
    callback: (response) => callback(response.credential),
  });

  window.google.accounts.id.renderButton(element, {
    theme: 'outline',
    size: 'large',
    width: 320,
    text: 'continue_with',
  });
}
