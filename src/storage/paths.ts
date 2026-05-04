import { homedir, platform } from 'node:os';
import { join } from 'node:path';
import { readEnv } from '../env.ts';

export const configDir = (env: NodeJS.ProcessEnv = process.env): string => {
  const e = readEnv(env);
  if (e.configDir) return e.configDir;
  if (platform() === 'win32') {
    const appData = env.APPDATA ?? join(homedir(), 'AppData', 'Roaming');
    return join(appData, 'oriyn');
  }
  const xdg = env.XDG_CONFIG_HOME ?? join(homedir(), '.config');
  return join(xdg, 'oriyn');
};

export const credentialsPath = (env?: NodeJS.ProcessEnv): string =>
  join(configDir(env), 'credentials.json');

export const cliConfigPath = (env?: NodeJS.ProcessEnv): string =>
  join(configDir(env), 'config.json');

// Project-link file lives at the nearest ancestor of cwd. Resolution lives in
// link/resolver.ts; this constant is the file's basename.
export const PROJECT_LINK_FILENAME = 'oriyn.json';
