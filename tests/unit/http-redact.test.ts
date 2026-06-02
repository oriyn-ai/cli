import { describe, expect, test } from 'bun:test';
import { redact, redactObject } from '../../src/http/redact.ts';

describe('http/redact', () => {
  test('redacts Bearer tokens', () => {
    expect(redact('Authorization: Bearer abc.def.ghi')).toContain('[REDACTED]');
    expect(redact('Authorization: Bearer abc.def.ghi')).not.toContain('abc.def.ghi');
  });

  test('redacts JWTs', () => {
    // Realistic JWT shape: each segment is base64url, > 20 chars.
    const jwt =
      'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c3IxMjMiLCJpYXQiOjE3MDAwMDAwMDB9.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';
    const out = redact(jwt);
    expect(out).toContain('[REDACTED]');
    expect(out).not.toContain('eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9');
  });

  test('redactObject walks nested structures', () => {
    const result = redactObject({
      headers: { authorization: 'Bearer abc.def.ghi' },
      meta: { code: 'state=foo' },
    });
    const json = JSON.stringify(result);
    expect(json).toContain('[REDACTED]');
    expect(json).not.toContain('abc.def.ghi');
  });

  test('redactObject redacts sensitive field names', () => {
    const result = redactObject({
      access_token: 'plain-token-value',
      nested: { secret_key: 'plain-secret-value' },
    });

    expect(result.access_token).toBe('[REDACTED]');
    expect(result.nested.secret_key).toBe('[REDACTED]');
  });

  test('redactObject preserves structured error codes', () => {
    const result = redactObject({
      code: 'research_failed',
      exit: 2,
      meta: { code: 'api_500' },
    });

    expect(result.code).toBe('research_failed');
    expect(result.meta.code).toBe('api_500');
  });
});
