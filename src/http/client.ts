import ky, { type HTTPError, type KyInstance } from 'ky';
import type { AuthStore } from '../auth/store.ts';
import { ApiError, NetworkError, PermissionError } from './errors.ts';

const USER_AGENT = `oriyn-cli/${process.env.npm_package_version ?? 'dev'} bun/${Bun.version}`;

export interface HttpClientOptions {
  apiBase: string;
  auth: AuthStore;
  /** Optional token override that bypasses the AuthStore (used for unauthenticated calls). */
  unauthenticated?: boolean;
}

const parseErrorBody = async (response: Response): Promise<{ message: string; body: unknown }> => {
  try {
    const text = await response.clone().text();
    if (!text) return { message: response.statusText, body: null };
    try {
      const json = JSON.parse(text) as Record<string, unknown>;
      const message =
        typeof json.error === 'string'
          ? json.error
          : typeof json.message === 'string'
            ? json.message
            : response.statusText;
      return { message, body: json };
    } catch {
      return { message: text.slice(0, 200), body: text };
    }
  } catch {
    return { message: response.statusText, body: null };
  }
};

export const createHttpClient = (opts: HttpClientOptions): KyInstance => {
  let refreshAttempted = false;
  return ky.create({
    prefixUrl: `${opts.apiBase.replace(/\/$/, '')}/v1`,
    headers: { 'user-agent': USER_AGENT },
    timeout: 60_000,
    retry: {
      limit: 2,
      methods: ['get', 'put', 'head', 'delete', 'options', 'trace'],
      statusCodes: [408, 429, 500, 502, 503, 504],
    },
    hooks: {
      beforeRequest: [
        async (request) => {
          if (opts.unauthenticated) return;
          const token = await opts.auth.getValidAccessToken();
          request.headers.set('authorization', `Bearer ${token}`);
        },
      ],
      afterResponse: [
        async (request, _opts, response) => {
          if (response.status === 401 && !refreshAttempted && !opts.unauthenticated) {
            refreshAttempted = true;
            // Force a refresh by clearing in-memory cache; getValidAccessToken
            // will refresh on next call.
            const token = await opts.auth.getValidAccessToken();
            request.headers.set('authorization', `Bearer ${token}`);
            return ky(request);
          }
          return response;
        },
      ],
      beforeError: [
        async (error: HTTPError) => {
          const { response } = error;
          if (!response) {
            throw new NetworkError(error.message, { cause: error });
          }
          const { message, body } = await parseErrorBody(response);
          if (response.status === 403 && body && typeof body === 'object') {
            const requiredPermission = (body as Record<string, unknown>).required_permission;
            if (typeof requiredPermission === 'string') {
              throw new PermissionError(message, response.status, body, requiredPermission);
            }
          }
          throw new ApiError(message, response.status, body);
        },
      ],
    },
  });
};
