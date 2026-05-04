import type { Command } from 'commander';
import { createApp } from '../app.ts';
import { reportAndExit } from '../lib/handle-error.ts';
import { resolveProduct } from '../link/resolver.ts';
import { ui } from '../output/human.ts';
import { writeJson } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';
import { configDir, credentialsPath } from '../storage/paths.ts';
import { loadCliConfig } from '../telemetry/config.ts';

export const registerStatus = (program: Command): void => {
  program
    .command('status')
    .description('One-screen diagnostic: auth, link, api, telemetry, paths')
    .action(async () => {
      const app = await createApp();
      try {
        const creds = await app.auth.load();
        const link = await resolveProduct({ cwd: app.cwd });
        const cfg = await loadCliConfig();

        let me: Awaited<ReturnType<typeof app.api.me>> | null = null;
        let apiReachable = false;
        if (creds || app.env.accessTokenOverride) {
          try {
            me = await app.api.me();
            apiReachable = true;
          } catch {
            apiReachable = false;
          }
        }

        const data = {
          auth: {
            logged_in: Boolean(creds || app.env.accessTokenOverride),
            email: me?.email ?? null,
            user_id: me?.user_id ?? null,
            token_source: app.env.accessTokenOverride ? 'env' : creds ? 'file' : null,
          },
          link: link ? { ...link } : null,
          api: { base: app.env.apiBase, reachable: apiReachable },
          telemetry: {
            enabled: cfg.telemetry.enabled !== false,
            device_id: cfg.telemetry.device_id,
          },
          paths: {
            config_dir: configDir(),
            credentials: credentialsPath(),
          },
        };

        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data });
          return;
        }

        const ok = ui.green(ui.check());
        const bad = ui.red(ui.cross());
        const out = process.stdout;
        out.write(`${ui.bold('oriyn status')}\n\n`);
        out.write(
          `${data.auth.logged_in ? ok : bad} auth: ${data.auth.email ?? (data.auth.logged_in ? 'logged in' : 'not logged in')}\n`,
        );
        out.write(
          `${link ? ok : bad} link: ${link ? `${link.productId} (${link.source})` : 'no project linked'}\n`,
        );
        out.write(`${apiReachable ? ok : bad} api: ${data.api.base}\n`);
        out.write(
          `${ok} telemetry: ${data.telemetry.enabled ? 'on' : 'off'} ${ui.dim(`(${data.telemetry.device_id ?? 'no device id'})`)}\n`,
        );
        out.write(`${ui.dim(`config: ${data.paths.config_dir}`)}\n`);
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerStatus;
