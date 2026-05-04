import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { renderTable, ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

export const registerProducts = (program: Command): void => {
  program
    .command('products')
    .description('List products in your org')
    .action(async () => {
      const app = await createApp();
      try {
        const products = await app.api.listProducts();
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: products });
          return;
        }
        if (products.length === 0) {
          process.stdout.write(`${ui.dim('No products yet. Create one in the web app.')}\n`);
          return;
        }
        process.stdout.write(
          `${renderTable(products, [
            { header: 'ID', render: (p) => p.id },
            { header: 'NAME', render: (p) => p.name },
            { header: 'STATUS', render: (p) => p.context_status },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerProducts;
