import { loadScript } from './loadScript';

let initialized = false;

export async function initFacebookSdk(appId: string): Promise<void> {
  if (initialized) return;

  await new Promise<void>((resolve) => {
    window.fbAsyncInit = () => {
      window.FB!.init({
        appId,
        cookie: true,
        xfbml: false,
        version: 'v21.0',
      });
      resolve();
    };

    loadScript('https://connect.facebook.net/en_US/sdk.js');
  });

  initialized = true;
}
