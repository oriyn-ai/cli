import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

const formatExpiresIn = (seconds: number): string => {
  if (seconds <= 0) return 'expired';
  if (seconds < 90) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  if (seconds < 86_400) return `${Math.round(seconds / 3600)}h`;
  return `${Math.round(seconds / 86_400)}d`;
};

export const registerStatus = (auth: Command): void => {
  auth
    .command('status')
    .description('Show whether you are logged in and when the access token expires')
    .action(async () => {
      const app = await createApp();
      try {
        const creds = await app.auth.load();
        const now = Math.floor(Date.now() / 1000);
        const data = creds
          ? {
              logged_in: true,
              expires_at: creds.expires_at,
              expires_in_seconds: creds.expires_at - now,
              source: app.env.accessTokenOverride ? 'env' : 'file',
            }
          : { logged_in: false };
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data });
          return;
        }
        if (!creds) {
          process.stdout.write(
            `${ui.red(ui.cross())} Not logged in. Run ${ui.cyan('oriyn auth login')}.\n`,
          );
          return;
        }
        const remaining = creds.expires_at - now;
        const status = remaining > 60 ? ui.green(ui.check()) : ui.yellow('!');
        process.stdout.write(
          `${status} Logged in. Token ${remaining > 0 ? `expires in ${formatExpiresIn(remaining)}` : 'expired (will refresh on next call)'}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};
