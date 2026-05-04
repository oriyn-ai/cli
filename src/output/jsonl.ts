import { redactObject } from '../http/redact.ts';

export type StreamEvent =
  | { type: 'step'; name: string; message?: string }
  | { type: 'progress'; pct?: number; message?: string }
  | { type: 'info'; message: string; data?: unknown }
  | { type: 'warn'; message: string; data?: unknown }
  | { type: 'error'; message: string; code?: string; data?: unknown }
  | { type: 'result'; data: unknown };

export interface JsonlEmitter {
  emit: (event: StreamEvent) => void;
  result: (data: unknown) => void;
  step: (name: string, message?: string) => void;
  progress: (pct: number, message?: string) => void;
  warn: (message: string, data?: unknown) => void;
  error: (message: string, code?: string, data?: unknown) => void;
}

export const createJsonlEmitter = (stream: NodeJS.WriteStream = process.stdout): JsonlEmitter => {
  const emit = (event: StreamEvent): void => {
    const safe = redactObject({ ...event, ts: new Date().toISOString() });
    stream.write(`${JSON.stringify(safe)}\n`);
  };
  return {
    emit,
    result: (data) => emit({ type: 'result', data }),
    step: (name, message) => emit({ type: 'step', name, ...(message ? { message } : {}) }),
    progress: (pct, message) => emit({ type: 'progress', pct, ...(message ? { message } : {}) }),
    warn: (message, data) => emit({ type: 'warn', message, ...(data ? { data } : {}) }),
    error: (message, code, data) =>
      emit({
        type: 'error',
        message,
        ...(code ? { code } : {}),
        ...(data ? { data } : {}),
      }),
  };
};

export const writeJson = (data: unknown, stream: NodeJS.WriteStream = process.stdout): void => {
  stream.write(`${JSON.stringify(redactObject(data), null, 2)}\n`);
};
