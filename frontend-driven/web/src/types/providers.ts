export {};

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (config: {
            client_id: string;
            callback: (response: { credential: string }) => void;
            auto_select?: boolean;
          }) => void;
          renderButton: (
            parent: HTMLElement,
            config: {
              theme?: string;
              size?: string;
              width?: number;
              text?: string;
            }
          ) => void;
          prompt: () => void;
        };
      };
    };
    fbAsyncInit?: () => void;
    FB?: {
      init: (config: {
        appId: string;
        cookie?: boolean;
        xfbml?: boolean;
        version: string;
      }) => void;
      login: (
        callback: (response: {
          authResponse?: { accessToken: string };
          status: string;
        }) => void,
        options?: { scope: string }
      ) => void;
    };
  }
}
