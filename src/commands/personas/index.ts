import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { requireProduct } from '../../lib/require-product.ts';
import { renderTable, ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

export const registerPersonas = (program: Command): void => {
  program
    .command('personas [id]')
    .description('List personas (no id) or show one persona in detail (with id)')
    .option('--product <id>', 'override linked product id')
    .action(async (id: string | undefined, opts: { product?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        if (id) {
          const profile = await app.api.getPersonaProfile(productId, id);
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: { id, ...profile } });
            return;
          }
          process.stdout.write(`${ui.bold('Persona')}: ${id}\n\n`);
          process.stdout.write(`${ui.bold('Static facts')}\n`);
          for (const f of profile.static_facts) process.stdout.write(`  ${ui.arrow()} ${f}\n`);
          process.stdout.write(`\n${ui.bold('Dynamic facts')}\n`);
          for (const f of profile.dynamic_facts) process.stdout.write(`  ${ui.arrow()} ${f}\n`);
          return;
        }

        const { enrichmentStatus, data } = await app.api.listPersonas(productId);
        if (resolveMode() === 'jsonl') {
          writeJson({
            type: 'result',
            data: { enrichment_status: enrichmentStatus, personas: data },
          });
          return;
        }
        if (data.length === 0) {
          process.stdout.write(
            `${ui.dim(`No personas yet (status: ${enrichmentStatus}). Run \`oriyn sync\`.`)}\n`,
          );
          return;
        }
        process.stdout.write(
          `${renderTable(data, [
            { header: 'ID', render: (p) => p.id, width: 14 },
            { header: 'NAME', render: (p) => p.name },
            { header: 'SIZE %', render: (p) => `${p.size_estimate}` },
            { header: 'TRAITS', render: (p) => `${p.behavioral_traits.length}` },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerPersonas;
