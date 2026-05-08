const TERMINAL_EXPERIMENT_STATUSES = new Set([
  'complete',
  'completed',
  'succeeded',
  'failed',
  'cancelled',
  'archived',
]);

export const isTerminalExperimentStatus = (status: string): boolean =>
  TERMINAL_EXPERIMENT_STATUSES.has(status);
