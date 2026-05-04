import { join } from 'node:path';
import type { Command } from 'commander';
import { reportAndExit } from '../../lib/handle-error.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';
import { removeFile } from '../../storage/file.ts';
import { PROJECT_LINK_FILENAME } from '../../storage/paths.ts';

export const registerUnlink = (program: Command): void => {
  program
    .command('unlink')
    .description(`Delete ${PROJECT_LINK_FILENAME} from the current directory`)
    .action(async () => {
      try {
        const path = join(process.cwd(), PROJECT_LINK_FILENAME);
        await removeFile(path);
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: { ok: true } });
        } else {
          process.stdout.write(`${ui.green(ui.check())} Unlinked\n`);
        }
      } catch (err) {
        reportAndExit(err);
      }
    });
};
