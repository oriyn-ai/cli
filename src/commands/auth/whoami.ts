import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

export const registerWhoami = (auth: Command): void => {
  auth
    .command('whoami')
    .description('Show the logged-in account')
    .action(async () => {
      const app = await createApp();
      try {
        const me = await app.api.me();
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: me });
        } else {
          process.stdout.write(`${ui.bold(me.email)} ${ui.dim(`(${me.user_id})`)}\n`);
        }
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};
