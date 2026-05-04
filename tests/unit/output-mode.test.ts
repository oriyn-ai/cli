import { describe, expect, test } from 'bun:test';
import { resolveMode } from '../../src/output/mode.ts';

const fakeStream = (isTTY: boolean) =>
  ({ isTTY, write: () => true }) as unknown as NodeJS.WriteStream;

describe('output/mode', () => {
  test('forceHuman wins', () => {
    expect(resolveMode({ forceHuman: true, stream: fakeStream(false) })).toBe('human');
  });

  test('forceJsonl wins over TTY', () => {
    expect(resolveMode({ forceJsonl: true, stream: fakeStream(true) })).toBe('jsonl');
  });

  test('CI env forces jsonl', () => {
    expect(
      resolveMode({
        stream: fakeStream(true),
        env: { CI: 'true' } as NodeJS.ProcessEnv,
      }),
    ).toBe('jsonl');
  });

  test('non-TTY pipe → jsonl', () => {
    expect(resolveMode({ stream: fakeStream(false), env: {} as NodeJS.ProcessEnv })).toBe('jsonl');
  });

  test('TTY → human', () => {
    expect(resolveMode({ stream: fakeStream(true), env: {} as NodeJS.ProcessEnv })).toBe('human');
  });
});
