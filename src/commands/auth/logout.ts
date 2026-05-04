import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

export const registerLogout = (auth: Command): void => {
  auth
    .command('logout')
    .description('Remove stored credentials from this machine')
    .action(async () => {
      const app = await createApp();
      try {
        await app.auth.clear();
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: { ok: true } });
        } else {
          process.stdout.write(`${ui.green(ui.check())} Logged out\n`);
        }
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};
