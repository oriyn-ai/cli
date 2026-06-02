import * as Sentry from '@sentry/bun';
import { redactObject } from '../http/redact.ts';
import { COMMIT, VERSION } from '../version.ts';
import { ciAutoSkip } from './env.ts';

type CaptureContext = Parameters<typeof Sentry.captureException>[1];
type MessageContext = Parameters<typeof Sentry.captureMessage>[1];

let initialized = false;

export const initSentry = (env: NodeJS.ProcessEnv = process.env): boolean => {
  const dsn = env.ORIYN_SENTRY_DSN ?? '';
  if (!dsn) return false;
  if (VERSION === '0.0.0-dev') return false;
  if (ciAutoSkip(env)) return false;
  Sentry.init({
    dsn,
    release: VERSION,
    environment: 'production',
    tracesSampleRate: 0,
    sendDefaultPii: false,
    beforeSend(event) {
      return redactObject(event);
    },
  });
  Sentry.setTag('commit', COMMIT);
  initialized = true;
  return true;
};

export const captureException = (err: unknown, context?: CaptureContext): void => {
  try {
    if (!initialized) return;
    Sentry.captureException(err, context);
  } catch {
    /* never throw from telemetry */
  }
};

export const captureMessage = (message: string, context?: MessageContext): void => {
  try {
    if (!initialized) return;
    Sentry.captureMessage(message, context);
  } catch {
    /* never throw from telemetry */
  }
};

export const flushSentry = async (timeoutMs = 1000): Promise<void> => {
  try {
    if (!initialized) return;
    await Sentry.flush(timeoutMs);
  } catch {
    /* swallow */
  }
};
