import { describe, expect, test } from 'bun:test';
import { isTerminalExperimentStatus } from '../../src/commands/experiments/status.ts';

describe('commands/experiments status polling', () => {
  test('treats API complete status as terminal', () => {
    expect(isTerminalExperimentStatus('complete')).toBe(true);
    expect(isTerminalExperimentStatus('failed')).toBe(true);
  });

  test('keeps active statuses non-terminal', () => {
    expect(isTerminalExperimentStatus('processing')).toBe(false);
    expect(isTerminalExperimentStatus('queued')).toBe(false);
  });
});
