import { readEnv } from '../env.ts';

export type OutputMode = 'human' | 'jsonl';

export interface ModeOptions {
  forceHuman?: boolean;
  forceJsonl?: boolean;
  stream?: NodeJS.WriteStream;
  env?: NodeJS.ProcessEnv;
}

export const resolveMode = (opts: ModeOptions = {}): OutputMode => {
  if (opts.forceHuman) return 'human';
  if (opts.forceJsonl) return 'jsonl';
  const env = readEnv(opts.env);
  // Explicit env override wins over TTY detection so AI agents in TTY-claiming
  // shells can force JSONL.
  if (env.telemetry === 'log') {
    // unrelated; keep going
  }
  if (env.isCI) return 'jsonl';
  const stream = opts.stream ?? process.stdout;
  return stream.isTTY ? 'human' : 'jsonl';
};

export const useColor = (opts: ModeOptions = {}): boolean => {
  const env = readEnv(opts.env);
  if (env.forceColor) return true;
  if (env.noColor) return false;
  const stream = opts.stream ?? process.stdout;
  return Boolean(stream.isTTY);
};
