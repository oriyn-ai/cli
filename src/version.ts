// Replaced at build time by `bun build --define VERSION=...`. Default for `bun run` dev mode.
declare const __ORIYN_VERSION__: string | undefined;
declare const __ORIYN_COMMIT__: string | undefined;

export const VERSION = typeof __ORIYN_VERSION__ === 'string' ? __ORIYN_VERSION__ : '0.0.0-dev';
export const COMMIT = typeof __ORIYN_COMMIT__ === 'string' ? __ORIYN_COMMIT__ : 'unknown';
