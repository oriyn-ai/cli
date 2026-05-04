import type { Command } from 'commander';
import open from 'open';
import { createApp } from '../app.ts';
import { reportAndExit } from '../lib/handle-error.ts';
import { resolveProduct } from '../link/resolver.ts';
import { ui } from '../output/human.ts';
import { writeJson } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';

export const registerOpen = (program: Command): void => {
  program
    .command('open [resource]')
    .description('Open the Oriyn web app for the linked product')
    .option('--product <id>', 'override linked product id')
    .action(async (resource: string | undefined, opts: { product?: string }) => {
      const app = await createApp();
      try {
        const link = await resolveProduct({ flagProduct: opts.product, cwd: app.cwd });
        const base = app.env.webBase.replace(/\/$/, '');
        const url = link ? `${base}/${link.productId}${resource ? `/${resource}` : ''}` : base;
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: { url } });
        } else {
          process.stdout.write(`${ui.dim('Opening:')} ${url}\n`);
        }
        await open(url);
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerOpen;
