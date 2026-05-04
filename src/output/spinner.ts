import ora, { type Ora } from 'ora';
import { resolveMode } from './mode.ts';

export interface Spinner {
  start: (text?: string) => void;
  update: (text: string) => void;
  succeed: (text?: string) => void;
  fail: (text?: string) => void;
  stop: () => void;
}

const noop: Spinner = {
  start: () => {},
  update: () => {},
  succeed: () => {},
  fail: () => {},
  stop: () => {},
};

export const createSpinner = (text?: string): Spinner => {
  if (resolveMode() === 'jsonl') return noop;
  let instance: Ora | null = null;
  return {
    start: (t) => {
      instance = ora(t ?? text ?? '').start();
    },
    update: (t) => {
      if (instance) instance.text = t;
    },
    succeed: (t) => {
      instance?.succeed(t);
      instance = null;
    },
    fail: (t) => {
      instance?.fail(t);
      instance = null;
    },
    stop: () => {
      instance?.stop();
      instance = null;
    },
  };
};
