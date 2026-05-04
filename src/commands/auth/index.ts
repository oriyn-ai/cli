import type { Command } from 'commander';
import { registerLogin } from './login.ts';
import { registerLogout } from './logout.ts';
import { registerStatus } from './status.ts';
import { registerWhoami } from './whoami.ts';

export const registerAuth = (program: Command): void => {
  const auth = program.command('auth').description('Manage CLI authentication');
  registerLogin(auth);
  registerLogout(auth);
  registerWhoami(auth);
  registerStatus(auth);
};
