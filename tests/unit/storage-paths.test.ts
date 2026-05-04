import { describe, expect, test } from 'bun:test';
import { configDir } from '../../src/storage/paths.ts';

describe('storage/paths.configDir', () => {
  test('uses ORIYN_CONFIG_DIR override', () => {
    expect(configDir({ ORIYN_CONFIG_DIR: '/tmp/oriyn-test' })).toBe('/tmp/oriyn-test');
  });

  test('uses XDG_CONFIG_HOME on POSIX', () => {
    if (process.platform === 'win32') return;
    expect(configDir({ XDG_CONFIG_HOME: '/var/xdg', HOME: '/home/u' })).toBe('/var/xdg/oriyn');
  });

  test('falls back to ~/.config/oriyn on POSIX without XDG', () => {
    if (process.platform === 'win32') return;
    const home = process.env.HOME ?? '/root';
    expect(configDir({ HOME: home })).toContain('/oriyn');
  });
});
