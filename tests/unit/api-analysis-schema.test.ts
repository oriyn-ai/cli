import { describe, expect, test } from 'bun:test';
import {
  bottlenecksResponseSchema,
  hypothesesResponseSchema,
  personasResponseSchema,
  productDetailSchema,
} from '../../src/api/types.ts';

describe('api/types analysis schemas', () => {
  test('parses product detail analysis_status', () => {
    const parsed = productDetailSchema.parse({
      id: 'bf492991-121a-4537-b8be-f961e54f950a',
      name: 'Oriyn',
      context: null,
      context_status: 'ready',
      analysis_status: 'ready',
      created_at: '2026-05-13T17:03:54Z',
    });

    expect(parsed.analysis_status).toBe('ready');
  });

  test('rejects legacy product detail enrichment_status', () => {
    expect(() =>
      productDetailSchema.parse({
        id: 'bf492991-121a-4537-b8be-f961e54f950a',
        name: 'Oriyn',
        context: null,
        context_status: 'ready',
        enrichment_status: 'ready',
        created_at: '2026-05-13T17:03:54Z',
      }),
    ).toThrow();
  });

  test('parses personas response analysis_status', () => {
    const parsed = personasResponseSchema.parse({
      analysis_status: 'ready',
      data: [
        {
          id: 'd5119ead-32b1-4683-a931-f5157dbb8ef3',
          name: 'Activation Seeker',
          description: 'Needs fast proof of value.',
          behavioral_traits: ['Visits setup repeatedly'],
          size_estimate: 42,
          generated_at: '2026-05-07T00:00:00Z',
          status: 'active',
          updated_at: '2026-05-07T00:00:00Z',
        },
      ],
    });

    expect(parsed.analysis_status).toBe('ready');
    expect(parsed.data[0]?.name).toBe('Activation Seeker');
  });

  test('parses pattern responses without source-user identifiers', () => {
    const hypothesis = hypothesesResponseSchema.parse({
      analysis_status: 'ready',
      data: [
        {
          sequence: ['entry', 'activation'],
          rendered_sequence: ['Entry', 'Activation'],
          frequency: 12,
          user_count: 4,
          significance_pct: 100,
        },
      ],
    });
    const bottleneck = bottlenecksResponseSchema.parse({
      analysis_status: 'ready',
      data: [
        {
          sequence: ['checkout', 'payment'],
          rendered_sequence: ['Checkout', 'Payment'],
          traversals: 7,
          user_count: 3,
          avg_duration_seconds: 14.5,
        },
      ],
    });

    expect(hypothesis.data[0]?.sequence).toEqual(['entry', 'activation']);
    expect(bottleneck.data[0]?.sequence).toEqual(['checkout', 'payment']);
  });
});
