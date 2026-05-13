import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { requireProduct } from '../../lib/require-product.ts';
import { renderTable, truncate, ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

const readFileText = async (path: string): Promise<string> => {
  return await Bun.file(path).text();
};

export const registerEvidence = (program: Command): void => {
  const cmd = program.command('evidence').description('Manage evidence sources for a product');

  cmd
    .command('add')
    .description('Add evidence from text, a file, or a URL')
    .requiredOption(
      '--kind <kind>',
      'evidence kind, e.g. founder_note, survey, transcript, public_url',
    )
    .option('--product <id>', 'override linked product id')
    .option('--title <title>', 'evidence title')
    .option('--text <text>', 'inline evidence text')
    .option('--file <path>', 'file containing evidence text')
    .option('--url <url>', 'evidence URL')
    .option('--confidence <n>', 'confidence from 0 to 1', (value) => Number.parseFloat(value))
    .action(
      async (opts: {
        kind: string;
        product?: string;
        title?: string;
        text?: string;
        file?: string;
        url?: string;
        confidence?: number;
      }) => {
        const app = await createApp();
        try {
          const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
          const body = opts.text ?? (opts.file ? await readFileText(opts.file) : undefined);
          if (!body && !opts.url) throw new Error('Provide one of --text, --file, or --url');
          const title = opts.title ?? opts.file ?? opts.url ?? 'Evidence';
          const source = await app.api.createEvidenceSource(productId, {
            kind: opts.kind,
            title,
            ...(opts.url ? { uri: opts.url } : {}),
            ...(body ? { body } : {}),
            ...(opts.confidence !== undefined ? { confidence: opts.confidence } : {}),
          });
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: source });
            return;
          }
          process.stdout.write(`${ui.green('Added evidence')}: ${source.title}\n`);
          process.stdout.write(`${ui.dim(`id: ${source.id} · status: ${source.status}`)}\n`);
        } catch (err) {
          reportAndExit(err);
        } finally {
          await app.shutdown();
        }
      },
    );

  cmd
    .command('list')
    .description('List evidence sources')
    .option('--product <id>', 'override linked product id')
    .action(async (opts: { product?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        const sources = await app.api.listEvidenceSources(productId);
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: sources });
          return;
        }
        if (sources.length === 0) {
          process.stdout.write(`${ui.dim('No evidence sources yet.')}\n`);
          return;
        }
        process.stdout.write(
          `${renderTable(sources, [
            { header: 'ID', render: (s) => s.id, width: 12 },
            { header: 'KIND', render: (s) => s.kind, width: 16 },
            { header: 'STATUS', render: (s) => s.status, width: 10 },
            { header: 'TITLE', render: (s) => truncate(s.title, 60) },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerEvidence;
