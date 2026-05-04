import { randomUUID } from 'node:crypto';

export interface Identity {
  deviceId: string;
  sessionId: string;
}

export const newDeviceId = (): string => `dev_${randomUUID()}`;
export const newSessionId = (): string => `sess_${randomUUID()}`;
