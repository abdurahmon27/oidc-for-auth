import { PublicClientApplication, type Configuration } from '@azure/msal-browser';

let msalInstance: PublicClientApplication | null = null;

export async function getMsalInstance(clientId: string, tenant: string): Promise<PublicClientApplication> {
  if (msalInstance) return msalInstance;

  const config: Configuration = {
    auth: {
      clientId,
      authority: `https://login.microsoftonline.com/${tenant}`,
      redirectUri: window.location.origin,
    },
    cache: {
      cacheLocation: 'sessionStorage',
    },
  };

  msalInstance = new PublicClientApplication(config);
  await msalInstance.initialize();
  return msalInstance;
}
