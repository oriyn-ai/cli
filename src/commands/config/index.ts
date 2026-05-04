import type { Command } from 'commander';
import { reportAndExit } from '../../lib/handle-error.ts';
import { ui } from '../../output/human.ts';
import { writeJson } from '../../output/jsonl.ts';
import { resolveMode } from '../../output/mode.ts';
import { loadCliConfig, saveCliConfig, setTelemetry } from '../../telemetry/config.ts';

const VALID_KEYS = ['api_base', 'default_product', 'telemetry'] as const;
type ConfigKey = (typeof VALID_KEYS)[number];

const isKey = (k: string): k is ConfigKey => (VALID_KEYS as readonly string[]).includes(k);

export const registerConfig = (program: Command): void => {
  program
    .command('config [key] [value]')
    .description('Show or update CLI config')
    .action(async (key: string | undefined, value: string | undefined) => {
      try {
        if (!key) {
          const cfg = await loadCliConfig();
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: cfg });
            return;
          }
          process.stdout.write(`${JSON.stringify(cfg, null, 2)}\n`);
          return;
        }
        if (!isKey(key)) {
          throw new Error(`Unknown config key '${key}'. Valid: ${VALID_KEYS.join(', ')}.`);
        }
        if (value === undefined) {
          const cfg = await loadCliConfig();
          const out =
            key === 'telemetry'
              ? cfg.telemetry.enabled === false
                ? 'off'
                : 'on'
              : ((cfg as Record<string, unknown>)[key] ?? null);
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: { [key]: out } });
            return;
          }
          process.stdout.write(`${out ?? ''}\n`);
          return;
        }
        if (key === 'telemetry') {
          const enabled = !['off', 'false', '0', 'no'].includes(value.toLowerCase());
          await setTelemetry(enabled);
          if (resolveMode() === 'jsonl') {
            writeJson({ type: 'result', data: { telemetry: enabled ? 'on' : 'off' } });
          } else {
            process.stdout.write(
              `${ui.green(ui.check())} telemetry ${enabled ? 'enabled' : 'disabled'}\n`,
            );
          }
          return;
        }
        const cfg = await loadCliConfig();
        const next = { ...cfg, [key]: value };
        await saveCliConfig(next);
        if (resolveMode() === 'jsonl') {
          writeJson({ type: 'result', data: { [key]: value } });
        } else {
          process.stdout.write(`${ui.green(ui.check())} ${key} = ${value}\n`);
        }
      } catch (err) {
        reportAndExit(err);
      }
    });
};

export default registerConfig;
