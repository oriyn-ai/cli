import { PostHog } from 'posthog-node';
import { readEnv } from '../env.ts';
import { COMMIT, VERSION } from '../version.ts';
import { ensureDeviceId, loadCliConfig, saveCliConfig } from './config.ts';
import { ciAutoSkip } from './env.ts';
import type { CliEvent } from './events.ts';
import { newSessionId } from './identity.ts';

// Public PostHog Cloud project key for the CLI. Safe to embed.
const POSTHOG_KEY = process.env.ORIYN_POSTHOG_KEY ?? 'phc_oriyn_cli_placeholder';
const POSTHOG_HOST = 'https://us.i.posthog.com';

export interface TelemetryClient {
  track: (event: CliEvent) => void;
  shutdown: () => Promise<void>;
  enabled: boolean;
  /** Print a one-line silent-opt-in announcement on first use. */
  maybeAnnounce: () => Promise<void>;
}

const noop: TelemetryClient = {
  track: () => {},
  shutdown: async () => {},
  enabled: false,
  maybeAnnounce: async () => {},
};

export const createTelemetryClient = async (
  env: NodeJS.ProcessEnv = process.env,
): Promise<TelemetryClient> => {
  const e = readEnv(env);
  if (e.telemetry === 'off' || e.doNotTrack) return noop;
  if (e.telemetry === 'unset' && ciAutoSkip(env)) return noop;
  if (VERSION === '0.0.0-dev') return noop;

  const { config } = await ensureDeviceId(env);
  const deviceId = config.telemetry.device_id;
  if (!deviceId) return noop;
  if (config.telemetry.enabled === false) return noop;

  const posthog = new PostHog(POSTHOG_KEY, { host: POSTHOG_HOST, flushAt: 1 });
  const sessionId = newSessionId();
  const baseProps = {
    cli_version: VERSION,
    cli_commit: COMMIT,
    session_id: sessionId,
    runtime: 'bun',
    bun_version: Bun.version,
  };

  return {
    enabled: true,
    track: (event: CliEvent) => {
      try {
        posthog.capture({
          distinctId: deviceId,
          event: event.name,
          properties: { ...baseProps, ...event.props },
        });
      } catch {
        // never fail user-facing flows on telemetry
      }
    },
    shutdown: async () => {
      try {
        await posthog.shutdown();
      } catch {
        /* swallow */
      }
    },
    maybeAnnounce: async () => {
      if (config.telemetry.decided_at) return;
      const announced: typeof config = {
        ...config,
        telemetry: {
          ...config.telemetry,
          enabled: true,
          decided_at: new Date().toISOString(),
        },
      };
      await saveCliConfig(announced, env);
      // One-line stderr announcement, no interactive prompt.
      console.error(
        'oriyn collects anonymous CLI usage data. Disable with `oriyn config telemetry off`.',
      );
    },
  };
};

export const loadTelemetryEnabled = async (
  env: NodeJS.ProcessEnv = process.env,
): Promise<{ enabled: boolean; deviceId: string | null }> => {
  const config = await loadCliConfig(env);
  return {
    enabled: config.telemetry.enabled !== false,
    deviceId: config.telemetry.device_id,
  };
};
