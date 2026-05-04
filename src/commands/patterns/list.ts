import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { requireProduct } from '../../lib/require-product.ts';
import { renderTable, truncate, ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';

export const registerPatterns = (program: Command): void => {
  program
    .command('patterns')
    .description('List mined hypotheses and bottlenecks')
    .option('--product <id>', 'override linked product id')
    .option('--only <kind>', 'filter to "hypothesis" or "bottleneck"')
    .action(async (opts: { product?: string; only?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        let patterns = await app.api.listPatterns(productId);
        if (opts.only === 'hypothesis' || opts.only === 'bottleneck') {
          patterns = patterns.filter((p) => p.kind === opts.only);
        }
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: patterns });
          return;
        }
        if (patterns.length === 0) {
          process.stdout.write(`${ui.dim('No patterns yet. Run `oriyn sync` to mine them.')}\n`);
          return;
        }
        process.stdout.write(
          `${renderTable(patterns, [
            { header: 'KIND', render: (p) => p.kind, width: 11 },
            {
              header: 'SEQUENCE',
              render: (p) => truncate(p.rendered_sequence.join(' → '), 60),
            },
            { header: 'USERS', render: (p) => `${p.user_count}` },
            {
              header: 'METRIC',
              render: (p) =>
                p.kind === 'hypothesis'
                  ? `${p.significance_pct.toFixed(1)}%`
                  : `${p.avg_duration_seconds.toFixed(1)}s`,
            },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerPatterns;
