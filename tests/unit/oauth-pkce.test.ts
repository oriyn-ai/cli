import { describe, expect, test } from 'bun:test';
import { generatePkce, generateState } from '../../src/oauth/pkce.ts';

describe('oauth/pkce', () => {
  test('generatePkce returns a verifier of valid length and an S256 challenge', async () => {
    const { codeVerifier, codeChallenge } = await generatePkce();
    expect(codeVerifier.length).toBeGreaterThanOrEqual(43);
    expect(codeVerifier.length).toBeLessThanOrEqual(128);
    expect(codeChallenge).toMatch(/^[A-Za-z0-9_-]+$/);
    expect(codeChallenge).not.toEqual(codeVerifier);
  });

  test('generateState returns a non-empty random string', () => {
    const a = generateState();
    const b = generateState();
    expect(a.length).toBeGreaterThan(16);
    expect(a).not.toEqual(b);
  });
});
