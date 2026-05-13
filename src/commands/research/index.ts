import type { Command } from 'commander';
import { createApp } from '../../app.ts';
import { reportAndExit } from '../../lib/handle-error.ts';
import { poll } from '../../lib/poll.ts';
import { requireProduct } from '../../lib/require-product.ts';
import { renderTable, truncate, ui } from '../../output/human.ts';
import { createJsonlEmitter, writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';
import { createSpinner } from '../../output/spinner.ts';

type RunKind = 'interview' | 'ab_test' | 'delphi' | 'playtest';

const isTerminal = (status: string): boolean =>
  ['complete', 'failed', 'cancelled'].includes(status);

async function createAndFollowRun(
  kind: RunKind,
  opts: {
    product?: string;
    title?: string;
    config: Record<string, unknown>;
    participantRule?: Record<string, unknown>;
  },
) {
  const app = await createApp();
  const emitter = createJsonlEmitter(process.stdout);
  const spinner = createSpinner('Resolving product…');
  try {
    spinner.start();
    const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
    if (resolveMode() === 'jsonl') emitter.step('resolve-product', productId);

    spinner.update('Creating research run…');
    const created = await app.api.createResearchRun(productId, {
      kind,
      ...(opts.title ? { title: opts.title } : {}),
      config: opts.config,
      ...(opts.participantRule ? { participant_rule: opts.participantRule } : {}),
    });
    if (resolveMode() === 'jsonl') {
      emitter.emit({ type: 'info', message: 'research run created', data: created });
    } else {
      process.stdout.write(`${ui.dim('View:')} ${created.url}\n`);
    }

    spinner.update('Running research…');
    let run = await app.api.getResearchRun(productId, created.research_run_id);
    run = await poll({
      fn: () => app.api.getResearchRun(productId, created.research_run_id),
      done: (value) => isTerminal(value.status),
      onTick: (value) => {
        spinner.update(`Status: ${value.status}`);
        if (resolveMode() === 'jsonl') emitter.emit({ type: 'progress', message: value.status });
      },
    });

    if (run.status === 'failed') {
      spinner.fail('Research run failed');
      if (resolveMode() === 'jsonl') emitter.error('Research run failed', 'research_failed', run);
      process.exit(2);
    }
    spinner.succeed(`Research ${run.status}`);
    if (resolveMode() === 'jsonl') {
      emitter.result({ ...run, url: created.url });
      return;
    }
    process.stdout.write(`\n${ui.bold(run.title)}\n`);
    if (run.output_summary) {
      process.stdout.write(`${JSON.stringify(run.output_summary, null, 2)}\n`);
    }
    process.stdout.write(`\n${ui.dim('View:')} ${created.url}\n`);
  } catch (err) {
    spinner.fail();
    reportAndExit(err);
  } finally {
    await app.shutdown();
  }
}

export const registerResearch = (program: Command): void => {
  const cmd = program.command('research').description('Run evidence-backed research modes');

  cmd
    .command('modes')
    .description('List supported research modes')
    .action(async () => {
      const app = await createApp();
      try {
        const modes = await app.api.listResearchModes();
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: modes });
          return;
        }
        process.stdout.write(
          `${renderTable(modes, [
            { header: 'KIND', render: (m) => m.kind, width: 12 },
            { header: 'WORKFLOW', render: (m) => m.workflow_name, width: 26 },
            { header: 'QUEUE', render: (m) => m.task_queue },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });

  cmd
    .command('list')
    .description('List research runs')
    .option('--product <id>', 'override linked product id')
    .action(async (opts: { product?: string }) => {
      const app = await createApp();
      try {
        const { productId } = await requireProduct({ flagProduct: opts.product, cwd: app.cwd });
        const runs = await app.api.listResearchRuns(productId);
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: runs });
          return;
        }
        if (runs.length === 0) {
          process.stdout.write(`${ui.dim('No research runs yet.')}\n`);
          return;
        }
        process.stdout.write(
          `${renderTable(runs, [
            { header: 'ID', render: (r) => r.id, width: 12 },
            { header: 'KIND', render: (r) => r.kind, width: 10 },
            { header: 'STATUS', render: (r) => r.status, width: 10 },
            { header: 'TITLE', render: (r) => truncate(r.title, 60) },
          ])}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      } finally {
        await app.shutdown();
      }
    });

  cmd
    .command('interview')
    .requiredOption('--question <text>', 'interview question')
    .option('--persona <id>', 'persona id')
    .option('--product <id>', 'override linked product id')
    .action((opts: { question: string; persona?: string; product?: string }) =>
      createAndFollowRun('interview', {
        product: opts.product,
        config: {
          question: opts.question,
          ...(opts.persona ? { persona_id: opts.persona } : {}),
        },
        ...(opts.persona ? { participantRule: { persona_ids: [opts.persona] } } : {}),
      }),
    );

  cmd
    .command('ab-test')
    .requiredOption('--a <text>', 'variant A text')
    .requiredOption('--b <text>', 'variant B text')
    .requiredOption('--question <text>', 'test question')
    .option('--product <id>', 'override linked product id')
    .action((opts: { a: string; b: string; question: string; product?: string }) =>
      createAndFollowRun('ab_test', {
        product: opts.product,
        config: {
          question: opts.question,
          variants: [
            { label: 'A', content: opts.a },
            { label: 'B', content: opts.b },
          ],
        },
      }),
    );

  cmd
    .command('delphi')
    .requiredOption('--question <text>', 'workshop question')
    .option('--rounds <n>', 'number of rounds', (value) => Number.parseInt(value, 10), 3)
    .option('--product <id>', 'override linked product id')
    .action((opts: { question: string; rounds: number; product?: string }) =>
      createAndFollowRun('delphi', {
        product: opts.product,
        config: { question: opts.question, rounds: opts.rounds },
      }),
    );

  cmd
    .command('playtest')
    .requiredOption('--url <url>', 'public or staging URL')
    .requiredOption('--task <text>', 'task for each persona to complete')
    .requiredOption('--allow-domain <domain...>', 'allowed domains')
    .option('--product <id>', 'override linked product id')
    .action((opts: { url: string; task: string; allowDomain: string[]; product?: string }) =>
      createAndFollowRun('playtest', {
        product: opts.product,
        config: {
          url: opts.url,
          task: opts.task,
          allowed_domains: opts.allowDomain,
        },
      }),
    );
};

export default registerResearch;
