import type { Command } from 'commander';
import open from 'open';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { startCallbackServer } from '../../oauth/callback-server.ts';
import { buildAuthorizeUrl, exchangeCode } from '../../oauth/flow.ts';
import { generatePkce, generateState } from '../../oauth/pkce.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';
import { createSpinner } from '../../output/spinner.ts';

export const registerLogin = (auth: Command): void => {
  auth
    .command('login')
    .description('Log in via browser (OAuth 2.1 + PKCE)')
    .option('--no-browser', "don't auto-open the browser; print the URL instead")
    .action(async (opts: { browser: boolean }) => {
      const app = await createApp();
      try {
        const { codeVerifier, codeChallenge } = await generatePkce();
        const state = generateState();
        const cb = startCallbackServer({ expectedState: state });
        const url = buildAuthorizeUrl({
          codeChallenge,
          state,
          redirectUri: cb.redirectUri,
        });

        const mode = resolveMode();
        const spinner = createSpinner('Waiting for browser…');
        if (mode === 'human') {
          process.stderr.write(`${ui.dim(`Opening ${url.toString()}`)}\n`);
        }
        if (opts.browser) {
          await open(url.toString()).catch(() => {
            process.stderr.write(`Could not auto-open browser. Visit:\n${url.toString()}\n`);
          });
        } else {
          process.stderr.write(`Open this URL in your browser:\n${url.toString()}\n`);
        }
        spinner.start();

        const { code, state: returnedState } = await cb.result;
        spinner.update('Exchanging code for tokens…');
        const tokens = await exchangeCode({
          code,
          state: returnedState,
          codeVerifier,
          redirectUri: cb.redirectUri,
        });
        await app.auth.save({
          access_token: tokens.accessToken,
          refresh_token: tokens.refreshToken,
          expires_at: tokens.expiresAt,
        });
        spinner.update('Fetching account…');
        const me = await app.api.me().catch(() => null);
        spinner.succeed(me ? `Logged in as ${me.email}` : 'Logged in');

        if (mode === 'jsonl') {
          writeJson({
            type: 'result',
            data: {
              email: me?.email ?? null,
              user_id: me?.user_id ?? null,
              expires_at: tokens.expiresAt,
            },
          });
        } else {
          process.stderr.write(
            `\n${ui.dim('Next:')} ${ui.bold('cd into your project')} and run ${ui.cyan('oriyn link')}\n`,
          );
        }
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};
