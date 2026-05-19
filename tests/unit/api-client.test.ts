import { describe, expect, test } from 'bun:test';
import { ApiClient } from '../../src/api/client.ts';
import type { AuthStore } from '../../src/auth/store.ts';

const auth = {
  getValidAccessToken: async () => 'token',
} as unknown as AuthStore;

describe('ApiClient', () => {
  test('passes persona_count to persona generation endpoint', async () => {
    let receivedPath = '';
    let receivedBody: unknown;
    const originalFetch = globalThis.fetch;

    globalThis.fetch = (async (
      input: Parameters<typeof fetch>[0],
      init?: Parameters<typeof fetch>[1],
    ) => {
      const request = input instanceof Request ? input : new Request(input.toString(), init);
      receivedPath = new URL(request.url).pathname;
      receivedBody = await request.clone().json();
      expect(request.headers.get('authorization')).toBe('Bearer token');
      return Response.json({ status: 'pending', workflow_id: 'wf_1' }, { status: 202 });
    }) as typeof fetch;

    try {
      const client = new ApiClient({ apiBase: 'https://api.example.test', auth });
      const response = await client.generatePersonas('prod_1', 7);

      expect(response).toEqual({ status: 'pending', workflow_id: 'wf_1' });
      expect(receivedPath).toBe('/v1/products/prod_1/personas/generate');
      expect(receivedBody).toEqual({ persona_count: 7 });
    } finally {
      globalThis.fetch = originalFetch;
    }
  });
});
