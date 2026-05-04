const DEFAULT_API_BASE = 'https://api.oriyn.ai';
const DEFAULT_WEB_BASE = 'https://app.oriyn.ai';

const truthy = (v: string | undefined): boolean => {
  if (!v) return false;
  const lower = v.toLowerCase();
  return lower === '1' || lower === 'true' || lower === 'on' || lower === 'yes';
};

const falsy = (v: string | undefined): boolean => {
  if (!v) return false;
  const lower = v.toLowerCase();
  return lower === '0' || lower === 'false' || lower === 'off' || lower === 'no';
};

export type TelemetryMode = 'on' | 'off' | 'log' | 'unset';

const readTelemetryMode = (v: string | undefined): TelemetryMode => {
  if (!v) return 'unset';
  if (v.toLowerCase() === 'log') return 'log';
  if (truthy(v)) return 'on';
  if (falsy(v)) return 'off';
  return 'unset';
};

export interface OriynEnv {
  apiBase: string;
  webBase: string;
  configDir: string | undefined;
  accessTokenOverride: string | undefined;
  product: string | undefined;
  org: string | undefined;
  telemetry: TelemetryMode;
  doNotTrack: boolean;
  forceColor: boolean;
  noColor: boolean;
  isCI: boolean;
}

export const readEnv = (source: NodeJS.ProcessEnv = process.env): OriynEnv => ({
  apiBase: source.ORIYN_API_BASE || DEFAULT_API_BASE,
  webBase: source.ORIYN_WEB_BASE || DEFAULT_WEB_BASE,
  configDir: source.ORIYN_CONFIG_DIR || undefined,
  accessTokenOverride: source.ORIYN_ACCESS_TOKEN || undefined,
  product: source.ORIYN_PRODUCT || undefined,
  org: source.ORIYN_ORG || undefined,
  telemetry: readTelemetryMode(source.ORIYN_TELEMETRY),
  doNotTrack: truthy(source.DO_NOT_TRACK),
  forceColor: truthy(source.FORCE_COLOR),
  noColor: truthy(source.NO_COLOR),
  isCI: truthy(source.CI),
});

export const DEFAULTS = { apiBase: DEFAULT_API_BASE, webBase: DEFAULT_WEB_BASE } as const;
