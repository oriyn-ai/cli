import { afterEach, beforeEach, describe, expect, test } from 'bun:test';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { NotLoggedInError } from '../../src/auth/errors.ts';
import { AuthStore } from '../../src/auth/store.ts';

let tmpRoot: string;
let credentialsFile: string;

beforeEach(async () => {
  tmpRoot = await mkdtemp(join(tmpdir(), 'oriyn-auth-'));
  credentialsFile = join(tmpRoot, 'credentials.json');
});

afterEach(async () => {
  await rm(tmpRoot, { recursive: true, force: true });
});

describe('auth/store', () => {
  test('returns env override token without touching disk', async () => {
    const store = new AuthStore({
      env: { ORIYN_ACCESS_TOKEN: 'env_tok' } as NodeJS.ProcessEnv,
      path: credentialsFile,
    });
    expect(await store.getValidAccessToken()).toBe('env_tok');
  });

  test('throws NotLoggedInError when no credentials file exists', async () => {
    const store = new AuthStore({ env: {} as NodeJS.ProcessEnv, path: credentialsFile });
    await expect(store.getValidAccessToken()).rejects.toBeInstanceOf(NotLoggedInError);
  });

  test('returns cached access token when not near expiry', async () => {
    const store = new AuthStore({ env: {} as NodeJS.ProcessEnv, path: credentialsFile });
    const future = Math.floor(Date.now() / 1000) + 3600;
    await store.save({
      access_token: 'cached_tok',
      refresh_token: 'rt',
      expires_at: future,
    });
    expect(await store.getValidAccessToken()).toBe('cached_tok');
  });

  test('writes credentials file with mode 0600', async () => {
    if (process.platform === 'win32') return;
    const store = new AuthStore({ env: {} as NodeJS.ProcessEnv, path: credentialsFile });
    await store.save({
      access_token: 'a',
      refresh_token: 'r',
      expires_at: Math.floor(Date.now() / 1000) + 3600,
    });
    const stat = await Bun.file(credentialsFile).stat();
    expect(stat.mode & 0o777).toBe(0o600);
  });
});
