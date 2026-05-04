import { spawn } from 'node:child_process';
import type { Command } from 'commander';
import { reportAndExit } from '../lib/handle-error.ts';
import { ui } from '../output/human.ts';
import { writeJson } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';

export const registerUpgrade = (program: Command): void => {
  program
    .command('upgrade')
    .description('Upgrade the CLI via `bun add -g oriyn@latest`')
    .action(async () => {
      try {
        const args = ['add', '-g', 'oriyn@latest'];
        if (resolveMode() === 'human') {
          process.stdout.write(`${ui.dim(`$ bun ${args.join(' ')}`)}\n`);
        }
        const child = spawn('bun', args, { stdio: 'inherit' });
        await new Promise<void>((res, rej) => {
          child.on('close', (code) => {
            if (code === 0) res();
            else rej(new Error(`bun exited with code ${code}`));
          });
          child.on('error', rej);
        });
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: { ok: true } });
        }
      } catch (err) {
        reportAndExit(err);
      }
    });
};

export default registerUpgrade;
