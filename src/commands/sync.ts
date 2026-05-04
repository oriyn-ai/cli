import type { Command } from 'commander';
import { createApp } from '../app.ts';
import { reportAndExit } from '../lib/handle-error.ts';
import { poll } from '../lib/poll.ts';
import { requireProduct } from '../lib/require-product.ts';
import { ui } from '../output/human.ts';
import { createJsonlEmitter } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';
import { createSpinner } from '../output/spinner.ts';

const READY = 'ready';

export const registerSync = (program: Command): void => {
  program
    .command('sync')
    .description('Run synthesis and enrichment as needed (idempotent)')
    .option('--product <id>', 'override linked product id')
    .option('--only <stage>', '"synthesize" or "enrich" — skip the other')
    .action(async (opts: { product?: string; only?: 'synthesize' | 'enrich' }) => {
      const app = await createApp();
      const spinner = createSpinner();
      const emitter = createJsonlEmitter(process.stdout);
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        spinner.start('Inspecting product…');
        let detail = await app.api.getProduct(productId);

        const wantSynthesis = opts.only !== 'enrich' && detail.context_status !== READY;
        if (wantSynthesis) {
          spinner.update('Starting synthesis…');
          if (resolveMode() === 'jsonl') emitter.step('synthesize');
          await app.api.startSynthesis(productId);
          detail = await poll({
            fn: () => app.api.getProduct(productId),
            done: (v) => v.context_status === READY || v.context_status === 'failed',
            onTick: (v) => {
              spinner.update(`Synthesis: ${v.context_status}`);
              if (resolveMode() === 'jsonl') {
                emitter.emit({ type: 'progress', message: `synthesis: ${v.context_status}` });
              }
            },
          });
          if (detail.context_status !== READY) {
            throw new Error(`Synthesis failed (status: ${detail.context_status})`);
          }
        }

        const wantEnrichment = opts.only !== 'synthesize' && detail.enrichment_status !== READY;
        if (wantEnrichment) {
          spinner.update('Starting enrichment…');
          if (resolveMode() === 'jsonl') emitter.step('enrich');
          await app.api.startEnrichment(productId);
          detail = await poll({
            fn: () => app.api.getProduct(productId),
            done: (v) => v.enrichment_status === READY || v.enrichment_status === 'failed',
            onTick: (v) => {
              spinner.update(`Enrichment: ${v.enrichment_status}`);
              if (resolveMode() === 'jsonl') {
                emitter.emit({
                  type: 'progress',
                  message: `enrichment: ${v.enrichment_status}`,
                });
              }
            },
          });
          if (detail.enrichment_status !== READY) {
            throw new Error(`Enrichment failed (status: ${detail.enrichment_status})`);
          }
        }

        spinner.succeed('Sync complete');
        if (resolveMode() === 'jsonl') {
          emitter.result({
            context_status: detail.context_status,
            enrichment_status: detail.enrichment_status,
          });
        } else {
          process.stdout.write(
            `${ui.green(ui.check())} context=${detail.context_status} enrichment=${detail.enrichment_status}\n`,
          );
        }
      } catch (err) {
        spinner.fail();
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerSync;
