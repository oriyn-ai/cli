const TERMINAL_EXPERIMENT_STATUSES = new Set(['complete', 'failed']);

export const isTerminalExperimentStatus = (status: string): boolean =>
  TERMINAL_EXPERIMENT_STATUSES.has(status);
