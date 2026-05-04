import { join } from 'node:path';
import * as p from '@clack/prompts';
import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { decodeJwtPayload } from '../../lib/jwt.ts';
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
    .option('--org <id>', 'override org id (defaults to active org from your session)')
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

        // Resolve current account + active org for display. Backend scopes
        // products via the auth context, so we don't need to pick org per call;
        // we just want to show the user *which* org they're linking on behalf
        // of so a multi-org user doesn't accidentally link the wrong product.
        const accessToken = await app.auth.getValidAccessToken();
        const claims = decodeJwtPayload(accessToken);
        const me = await app.api.me().catch(() => null);
        const activeOrg = claims.org_slug ?? claims.org_id ?? null;

        if (!orgId) {
          orgId = claims.org_id ?? '';
        }

        if (!productId) {
          if (resolveMode() === 'jsonl') {
            throw new Error('Refusing interactive picker in non-TTY mode. Pass --product <id>.');
          }
          const products = await app.api.listProducts();
          if (products.length === 0) {
            throw new Error('No products available in this org. Create one in the web app first.');
          }

          p.intro(ui.bold('Link a product'));
          process.stderr.write(
            `${ui.dim('Account:')} ${me?.email ?? '(unknown)'}` +
              `${activeOrg ? `  ${ui.dim('Org:')} ${activeOrg}` : ''}\n` +
              `${ui.dim('Repo dir:')} ${app.cwd}\n\n`,
          );

          const choice = await p.select({
            message: 'Pick a product to link',
            options: products.map((prod) => ({
              value: prod.id,
              label: prod.name,
              hint: `${prod.context_status} · ${prod.id.slice(0, 8)}`,
            })),
          });
          if (p.isCancel(choice)) {
            p.cancel('Cancelled');
            process.exit(1);
          }
          productId = choice as string;

          if (products.length > 1 || !opts.force) {
            // Light confirmation step so the picker selection is auditable.
            const picked = products.find((prod) => prod.id === productId);
            const confirm = await p.confirm({
              message: `Link to ${ui.bold(picked?.name ?? productId)}?`,
              initialValue: true,
            });
            if (p.isCancel(confirm) || !confirm) {
              p.cancel('Cancelled');
              process.exit(1);
            }
          }
          p.outro('Linking…');
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
