import pkg from '../package.json' with { type: 'json' };

export const VERSION = pkg.version as string;
export const COMMIT = process.env.ORIYN_COMMIT ?? 'unknown';
