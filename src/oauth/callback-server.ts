import { CALLBACK_PATH, CALLBACK_TIMEOUT_MS } from './config.ts';

const SUCCESS_HTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>Logged in</title>
<style>body{font:16px/1.5 system-ui,sans-serif;max-width:36rem;margin:6rem auto;padding:0 1rem;color:#111}
h1{font-size:1.5rem;margin:0 0 .5rem}p{color:#555}</style>
</head><body><h1>You're logged in to Oriyn.</h1>
<p>You can close this tab and return to your terminal.</p></body></html>`;

const errorHtml = (message: string): string => `<!doctype html>
<html><head><meta charset="utf-8"><title>Login failed</title>
<style>body{font:16px/1.5 system-ui,sans-serif;max-width:36rem;margin:6rem auto;padding:0 1rem;color:#111}
h1{font-size:1.5rem;margin:0 0 .5rem;color:#b91c1c}p{color:#555}</style>
</head><body><h1>Login failed</h1><p>${message.replace(/[<>&]/g, '')}</p></body></html>`;

export interface CallbackResult {
  code: string;
  state: string;
}

export interface CallbackHandle {
  port: number;
  redirectUri: string;
  result: Promise<CallbackResult>;
  close: () => void;
}

export const startCallbackServer = (input: { expectedState: string }): CallbackHandle => {
  let resolve!: (value: CallbackResult) => void;
  let reject!: (err: Error) => void;
  const result = new Promise<CallbackResult>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  const server = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch(req) {
      const url = new URL(req.url);
      if (url.pathname !== CALLBACK_PATH) {
        return new Response('Not found', { status: 404 });
      }
      const error = url.searchParams.get('error');
      if (error) {
        const description = url.searchParams.get('error_description') ?? error;
        reject(new Error(`OAuth provider returned error: ${description}`));
        return new Response(errorHtml(description), {
          status: 400,
          headers: { 'content-type': 'text/html; charset=utf-8' },
        });
      }
      const code = url.searchParams.get('code');
      const state = url.searchParams.get('state');
      if (!code || !state) {
        reject(new Error('OAuth callback missing code or state'));
        return new Response(errorHtml('Missing code or state.'), {
          status: 400,
          headers: { 'content-type': 'text/html; charset=utf-8' },
        });
      }
      if (state !== input.expectedState) {
        reject(new Error('OAuth state mismatch (possible CSRF attempt)'));
        return new Response(errorHtml('State mismatch.'), {
          status: 400,
          headers: { 'content-type': 'text/html; charset=utf-8' },
        });
      }
      resolve({ code, state });
      return new Response(SUCCESS_HTML, {
        status: 200,
        headers: { 'content-type': 'text/html; charset=utf-8' },
      });
    },
  });

  const timeout = setTimeout(() => {
    reject(new Error('Login timed out waiting for browser callback'));
  }, CALLBACK_TIMEOUT_MS);

  const close = () => {
    clearTimeout(timeout);
    server.stop();
  };

  // Auto-close after either resolution or rejection.
  result.finally(close).catch(() => {});

  const port = server.port;
  if (port == null) {
    close();
    throw new Error('Failed to bind callback server');
  }

  return {
    port,
    redirectUri: `http://127.0.0.1:${port}${CALLBACK_PATH}`,
    result,
    close,
  };
};
