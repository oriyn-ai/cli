import { afterEach, beforeEach, describe, expect, test } from 'bun:test';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { findProjectLinkPath, resolveProduct } from '../../src/link/resolver.ts';

let tmpRoot: string;

beforeEach(async () => {
  tmpRoot = await mkdtemp(join(tmpdir(), 'oriyn-link-'));
});

afterEach(async () => {
  await rm(tmpRoot, { recursive: true, force: true });
});

describe('link/resolver', () => {
  test('findProjectLinkPath returns null when no oriyn.json above cwd', async () => {
    const sub = join(tmpRoot, 'a', 'b');
    await writeFile(join(tmpRoot, '.placeholder'), '', { mode: 0o644 });
    expect(await findProjectLinkPath(sub)).toBeNull();
  });

  test('findProjectLinkPath walks up to the nearest oriyn.json', async () => {
    const link = join(tmpRoot, 'oriyn.json');
    await writeFile(link, JSON.stringify({ orgId: 'o', productId: 'p' }));
    const sub = join(tmpRoot, 'apps', 'foo', 'src');
    expect(await findProjectLinkPath(sub)).toBe(link);
  });

  test('resolveProduct prefers --product flag', async () => {
    const r = await resolveProduct({ flagProduct: 'prod_X', cwd: tmpRoot });
    expect(r?.productId).toBe('prod_X');
    expect(r?.source).toBe('flag');
  });

  test('resolveProduct prefers ORIYN_PRODUCT env over file', async () => {
    const link = join(tmpRoot, 'oriyn.json');
    await writeFile(link, JSON.stringify({ orgId: 'o', productId: 'in_file' }));
    const r = await resolveProduct({
      cwd: tmpRoot,
      env: { ORIYN_PRODUCT: 'in_env' } as NodeJS.ProcessEnv,
    });
    expect(r?.productId).toBe('in_env');
    expect(r?.source).toBe('env');
  });

  test('resolveProduct returns null when nothing matches', async () => {
    const r = await resolveProduct({
      cwd: tmpRoot,
      env: {} as NodeJS.ProcessEnv,
    });
    expect(r).toBeNull();
  });
});
