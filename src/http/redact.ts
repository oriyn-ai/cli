// Strip bearer tokens and obvious secrets from arbitrary strings before they reach
// stderr, telemetry, or Sentry. Conservative — false positives are fine.
const PATTERNS: ReadonlyArray<RegExp> = [
  /Bearer\s+[A-Za-z0-9._~+/-]+=*/g,
  /eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}/g,
  /sk_(live|test)_[A-Za-z0-9]{20,}/g,
  /(refresh_token|access_token|code|state|code_verifier)=[A-Za-z0-9._~+/-]+/gi,
];

const SECRET_FIELDS = new Set([
  'apikey',
  'api_key',
  'authorization',
  'bearer',
  'code',
  'code_verifier',
  'credentials',
  'refresh_token',
  'access_token',
  'secret',
  'secret_key',
  'secret_value',
  'token',
]);

export const redact = (input: string): string => {
  let out = input;
  for (const pattern of PATTERNS) {
    out = out.replace(pattern, '[REDACTED]');
  }
  return out;
};

export const redactObject = <T>(value: T): T => {
  if (value == null) return value;
  if (typeof value === 'string') return redact(value) as unknown as T;
  if (Array.isArray(value)) return value.map(redactObject) as unknown as T;
  if (typeof value === 'object') {
    const result: Record<string, unknown> = {};
    for (const [key, v] of Object.entries(value as Record<string, unknown>)) {
      result[key] = SECRET_FIELDS.has(key.toLowerCase()) ? '[REDACTED]' : redactObject(v);
    }
    return result as T;
  }
  return value;
};
