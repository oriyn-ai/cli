import { ApiClient } from './api/client.ts';
import { AuthStore } from './auth/store.ts';
import { type OriynEnv, readEnv } from './env.ts';
import { createTelemetryClient, type TelemetryClient } from './telemetry/client.ts';

export interface App {
  env: OriynEnv;
  auth: AuthStore;
  api: ApiClient;
  telemetry: TelemetryClient;
  cwd: string;
  shutdown: () => Promise<void>;
}

export interface AppOptions {
  apiBaseOverride?: string;
  cwd?: string;
}

export const createApp = async (opts: AppOptions = {}): Promise<App> => {
  const env = readEnv();
  const apiBase = opts.apiBaseOverride ?? env.apiBase;
  const auth = new AuthStore();
  const api = new ApiClient({ apiBase, auth });
  const telemetry = await createTelemetryClient();
  const cwd = opts.cwd ?? process.cwd();

  const shutdown = async () => {
    await telemetry.shutdown();
  };

  return { env: { ...env, apiBase }, auth, api, telemetry, cwd, shutdown };
};
