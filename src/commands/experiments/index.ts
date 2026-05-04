import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { poll } from '../../lib/poll.ts';
import { requireProduct } from '../../lib/require-product.ts';
import { renderTable, truncate, ui } from '../../output/human.ts';
import { createJsonlEmitter, writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';
import { createSpinner } from '../../output/spinner.ts';

const TERMINAL_STATUSES = new Set(['completed', 'succeeded', 'failed', 'cancelled', 'archived']);

export const registerExperiments = (program: Command): void => {
  const cmd = program
    .command('experiments [id]')
    .description('List experiments (no id) or show one experiment in detail')
    .option('--product <id>', 'override linked product id')
    .action(async (id: string | undefined, opts: { product?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        if (id) {
          const exp = await app.api.getExperiment(productId, id);
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: exp });
          } else {
            process.stdout.write(`${ui.bold(exp.hypothesis)}\n`);
            process.stdout.write(
              `${ui.dim(`status: ${exp.status} · created: ${exp.created_at}`)}\n`,
            );
            if (exp.summary) {
              process.stdout.write(
                `\n${ui.bold('Verdict')}: ${exp.summary.verdict} (${(exp.summary.convergence * 100).toFixed(0)}% convergence)\n`,
              );
              process.stdout.write(`${exp.summary.summary}\n`);
            }
          }
          return;
        }
        const list = await app.api.listExperiments(productId);
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: list });
          return;
        }
        if (list.length === 0) {
          process.stdout.write(`${ui.dim('No experiments yet.')}\n`);
          return;
        }
        process.stdout.write(
          `${renderTable(list, [
            { header: 'ID', render: (e) => e.id, width: 12 },
            { header: 'STATUS', render: (e) => e.status, width: 11 },
            { header: 'VERDICT', render: (e) => e.verdict ?? '—', width: 9 },
            { header: 'HYPOTHESIS', render: (e) => truncate(e.hypothesis, 60) },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });

  cmd
    .command('run <hypothesis>')
    .description('Run an experiment against the linked product (streams progress)')
    .option('--product <id>', 'override linked product id')
    .option('--agents <n>', 'number of simulated agents', (v) => Number.parseInt(v, 10))
    .action(async (hypothesis: string, opts: { product?: string; agents?: number }) => {
      const app = await createApp();
      const emitter = createJsonlEmitter(process.stdout);
      const spinner = createSpinner('Resolving product…');
      try {
        spinner.start();
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        if (resolveMode() === 'jsonl') emitter.step('resolve-product', productId);

        spinner.update('Creating experiment…');
        if (resolveMode() === 'jsonl') emitter.step('create-experiment');
        const created = await app.api.createExperiment(productId, {
          hypothesis,
          ...(opts.agents ? { agent_count: opts.agents } : {}),
        });

        if (resolveMode() === 'jsonl') {
          emitter.emit({
            type: 'info',
            message: 'experiment created',
            data: { id: created.experimentId, url: created.url },
          });
        } else {
          process.stdout.write(`${ui.dim('View:')} ${created.url}\n`);
        }

        spinner.update('Running simulation…');
        let last = await app.api.getExperiment(productId, created.experimentId);
        if (resolveMode() === 'jsonl') {
          emitter.emit({ type: 'progress', message: `status: ${last.status}` });
        }
        last = await poll({
          fn: () => app.api.getExperiment(productId, created.experimentId),
          done: (v) => TERMINAL_STATUSES.has(v.status),
          onTick: (v) => {
            spinner.update(`Status: ${v.status}`);
            if (resolveMode() === 'jsonl') {
              emitter.emit({ type: 'progress', message: `status: ${v.status}` });
            }
          },
        });

        if (last.status === 'failed') {
          spinner.fail('Experiment failed');
          if (resolveMode() === 'jsonl') {
            emitter.error('Experiment failed', 'experiment_failed', { id: last.id });
          }
          process.exit(2);
        }

        spinner.succeed(
          last.summary
            ? `Verdict: ${last.summary.verdict} (${(last.summary.convergence * 100).toFixed(0)}%)`
            : 'Done',
        );

        if (resolveMode() === 'jsonl') {
          emitter.result({ ...last, url: created.url });
          return;
        }
        if (last.summary) {
          process.stdout.write(`\n${ui.bold('Summary')}\n${last.summary.summary}\n\n`);
          process.stdout.write(`${ui.bold('By persona')}\n`);
          for (const row of last.summary.persona_breakdown) {
            process.stdout.write(
              `  ${ui.cyan(row.persona)} ${ui.dim(`adoption ${(row.adoption_rate * 100).toFixed(0)}%`)}: ${truncate(row.response, 80)}\n`,
            );
          }
        }
        process.stdout.write(`\n${ui.dim('View:')} ${created.url}\n`);
      } catch (err) {
        spinner.fail();
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });
};

export default registerExperiments;
