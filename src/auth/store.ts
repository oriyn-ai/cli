import { readEnv } from '../env.ts';
import { refreshTokens } from '../oauth/flow.ts';
import { readJson, removeFile, writeJsonAtomic } from '../storage/file.ts';
import { credentialsPath } from '../storage/paths.ts';
import {
  CREDENTIALS_SCHEMA_VERSION,
  credentialsSchema,
  type StoredCredentials,
} from './credentials.ts';
import { NotLoggedInError, SessionExpiredError } from './errors.ts';

const REFRESH_SKEW_SECONDS = 60;

export interface AuthStoreOptions {
  env?: NodeJS.ProcessEnv;
  /** Override path for tests. */
  path?: string;
}

export class AuthStore {
  private readonly env: NodeJS.ProcessEnv;
  private readonly path: string;
  private inflight: Promise<string> | null = null;

  constructor(opts: AuthStoreOptions = {}) {
    this.env = opts.env ?? process.env;
    this.path = opts.path ?? credentialsPath(this.env);
  }

  async load(): Promise<StoredCredentials | null> {
    const raw = await readJson<unknown>(this.path);
    if (!raw) return null;
    const parsed = credentialsSchema.safeParse(raw);
    if (!parsed.success) {
      // Corrupt or stale schema — wipe so next login gets a clean slate.
      await removeFile(this.path);
      return null;
    }
    return parsed.data;
  }

  async save(creds: Omit<StoredCredentials, 'schema_version'>): Promise<void> {
    const payload: StoredCredentials = { schema_version: CREDENTIALS_SCHEMA_VERSION, ...creds };
    await writeJsonAtomic(this.path, payload, { mode: 0o600 });
  }

  async clear(): Promise<void> {
    await removeFile(this.path);
  }

  /**
   * Returns a valid access token, refreshing if needed. Throws NotLoggedInError
   * or SessionExpiredError on failure. Honours ORIYN_ACCESS_TOKEN as a CI
   * escape hatch — that path skips the file entirely.
   */
  async getValidAccessToken(): Promise<string> {
    const env = readEnv(this.env);
    if (env.accessTokenOverride) return env.accessTokenOverride;
    if (this.inflight) return this.inflight;
    this.inflight = this.resolveAccessToken().finally(() => {
      this.inflight = null;
    });
    return this.inflight;
  }

  private async resolveAccessToken(): Promise<string> {
    const creds = await this.load();
    if (!creds) throw new NotLoggedInError();
    const now = Math.floor(Date.now() / 1000);
    if (creds.expires_at - now > REFRESH_SKEW_SECONDS) return creds.access_token;
    try {
      const fresh = await refreshTokens(creds.refresh_token);
      await this.save({
        access_token: fresh.accessToken,
        refresh_token: fresh.refreshToken,
        expires_at: fresh.expiresAt,
      });
      return fresh.accessToken;
    } catch {
      await this.clear();
      throw new SessionExpiredError();
    }
  }
}
