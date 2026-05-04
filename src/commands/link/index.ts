import { join } from 'node:path';
import * as p from '@clack/prompts';
import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { writeProjectLink } from '../../link/file.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';
import { PROJECT_LINK_FILENAME } from '../../storage/paths.ts';
import { registerUnlink } from './unlink.ts';

export const registerLink = (program: Command): void => {
  program
    .command('link')
    .description('Link this directory to an Oriyn product (writes oriyn.json)')
    .option('--product <id>', 'pick a specific product without prompting')
    .option('--org <id>', 'override org id when product is ambiguous')
    .option('--force', 'overwrite an existing oriyn.json')
    .action(async (opts: { product?: string; org?: string; force?: boolean }) => {
      const app = await createApp();
      try {
        const linkPath = join(app.cwd, PROJECT_LINK_FILENAME);
        const exists = await Bun.file(linkPath).exists();
        if (exists && !opts.force) {
          throw new Error(
            `${PROJECT_LINK_FILENAME} already exists in this directory. Pass --force to overwrite.`,
          );
        }

        let productId = opts.product;
        let orgId = opts.org;

        if (!productId) {
          if (resolveMode() === 'jsonl') {
            throw new Error('Refusing interactive picker in non-TTY mode. Pass --product <id>.');
          }
          const products = await app.api.listProducts();
          if (products.length === 0) {
            throw new Error('No products available. Create one in the web app first.');
          }
          if (products.length === 1) {
            productId = products[0]!.id;
          } else {
            const choice = await p.select({
              message: 'Pick a product to link',
              options: products.map((prod) => ({
                value: prod.id,
                label: prod.name,
                hint: prod.context_status,
              })),
            });
            if (p.isCancel(choice)) {
              p.cancel('Cancelled');
              process.exit(1);
            }
            productId = choice as string;
          }
        }

        if (!orgId) {
          // Resolve org id from /me when not explicitly set
          const me = await app.api.me().catch(() => null);
          // Backend currently scopes products by org via auth context;
          // passing the user_id keeps oriyn.json self-describing.
          orgId = me?.user_id ?? '';
        }

        await writeProjectLink(linkPath, { orgId, productId });

        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: { ok: true, path: linkPath, productId, orgId } });
        } else {
          process.stdout.write(
            `${ui.green(ui.check())} Linked ${ui.bold(productId)} → ${ui.dim(linkPath)}\n`,
          );
          process.stdout.write(
            `${ui.dim('Try:')} ${ui.cyan('oriyn experiments run "<your hypothesis>"')}\n`,
          );
        }
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });

  registerUnlink(program);
};

export default registerLink;
