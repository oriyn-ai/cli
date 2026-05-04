import * as Sentry from '@sentry/bun';
import { redactObject } from '../http/redact.ts';
import { COMMIT, VERSION } from '../version.ts';
import { ciAutoSkip } from './env.ts';

const PUBLIC_DSN = process.env.ORIYN_SENTRY_DSN ?? '';

export const initSentry = (env: NodeJS.ProcessEnv = process.env): boolean => {
  if (!PUBLIC_DSN) return false;
  if (VERSION === '0.0.0-dev') return false;
  if (ciAutoSkip(env)) return false;
  Sentry.init({
    dsn: PUBLIC_DSN,
    release: VERSION,
    environment: 'production',
    tracesSampleRate: 0,
    sendDefaultPii: false,
    beforeSend(event) {
      return redactObject(event);
    },
  });
  Sentry.setTag('commit', COMMIT);
  return true;
};

export const captureException = (err: unknown): void => {
  try {
    Sentry.captureException(err);
  } catch {
    /* never throw from telemetry */
  }
};

export const flushSentry = async (timeoutMs = 1000): Promise<void> => {
  try {
    await Sentry.flush(timeoutMs);
  } catch {
    /* swallow */
  }
};
