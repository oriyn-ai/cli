import { describe, expect, test } from 'bun:test';
import { createExperimentResponseSchema } from '../../src/api/types.ts';

describe('api/types createExperimentResponseSchema', () => {
  test('parses experiment_id and url', () => {
    const parsed = createExperimentResponseSchema.parse({
      experiment_id: 'exp-1',
      url: 'https://app.oriyn.ai/acme/dark-mode/experiments/exp-1',
    });
    expect(parsed.experiment_id).toBe('exp-1');
    expect(parsed.url).toBe('https://app.oriyn.ai/acme/dark-mode/experiments/exp-1');
  });

  test('rejects payload without url', () => {
    expect(() => createExperimentResponseSchema.parse({ experiment_id: 'exp-1' })).toThrow();
  });

  test('rejects non-URL url', () => {
    expect(() =>
      createExperimentResponseSchema.parse({ experiment_id: 'exp-1', url: 'not-a-url' }),
    ).toThrow();
  });
});
