import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { requireProduct } from '../../lib/require-product.ts';
import { renderTable, ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

export const registerPersonas = (program: Command): void => {
  const cmd = program
    .command('personas [id]')
    .description('List personas (no id) or show one persona in detail (with id)')
    .option('--product <id>', 'override linked product id')
    .action(async (id: string | undefined, opts: { product?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        if (id) {
          const { data } = await app.api.listPersonas(productId);
          const persona = data.find((item) => item.id === id);
          if (!persona) throw new Error(`Persona not found: ${id}`);
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: persona });
            return;
          }
          process.stdout.write(`${ui.bold('Persona')}: ${id}\n\n`);
          process.stdout.write(`${ui.bold(persona.name)}\n`);
          process.stdout.write(`${persona.description}\n\n`);
          process.stdout.write(`${ui.bold('Traits')}\n`);
          for (const trait of persona.behavioral_traits)
            process.stdout.write(`  ${ui.arrow()} ${trait}\n`);
          process.stdout.write(`\n${ui.bold('Size')}: ${persona.size_estimate}%\n`);
          return;
        }

        const { analysisStatus, data } = await app.api.listPersonas(productId);
        if (resolveMode() === 'jsonl') {
          writeJson({
            type: 'result',
            data: { analysis_status: analysisStatus, personas: data },
          });
          return;
        }
        if (data.length === 0) {
          process.stdout.write(
            `${ui.dim(`No personas yet (status: ${analysisStatus}). Run \`oriyn sync\`.`)}\n`,
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

  cmd
    .command('generate')
    .description('Generate behavior-grounded personas for the linked product')
    .option('--product <id>', 'override linked product id')
    .action(async (opts: { product?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        const started = await app.api.generatePersonas(productId);
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: started });
          return;
        }
        process.stdout.write(`${ui.green('Persona generation started')}\n`);
        process.stdout.write(`${ui.dim(`status: ${started.status}`)}\n`);
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerPersonas;
