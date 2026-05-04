#!/usr/bin/env bun
import { Command } from 'commander';
import { registerAuth } from './commands/auth/index.ts';
import registerCompletion from './commands/completion.ts';
import registerConfig from './commands/config/index.ts';
import registerExperiments from './commands/experiments/index.ts';
import registerLink from './commands/link/index.ts';
import registerOpen from './commands/open.ts';
import registerPatterns from './commands/patterns/list.ts';
import registerPersonas from './commands/personas/index.ts';
import registerProducts from './commands/products/list.ts';
import registerStatus from './commands/status.ts';
import registerSync from './commands/sync.ts';
import registerUpgrade from './commands/upgrade.ts';
import { reportAndExit } from './lib/handle-error.ts';
import { flushSentry, initSentry } from './telemetry/sentry.ts';
import { VERSION } from './version.ts';

initSentry();

const program = new Command();

program
  .name('oriyn')
  .description('Predict how users respond to product changes before shipping.')
  .version(VERSION, '-v, --version', 'output the version number')
  .helpOption('-h, --help', 'display help for command')
  .option('--api-base <url>', 'override Oriyn API base URL (env: ORIYN_API_BASE)')
  .option('--human', 'force human-readable output even when piped')
  .showHelpAfterError();

registerAuth(program);
registerLink(program);
registerProducts(program);
registerPersonas(program);
registerPatterns(program);
registerExperiments(program);
registerSync(program);
registerStatus(program);
registerConfig(program);
registerOpen(program);
registerUpgrade(program);
registerCompletion(program);

const main = async () => {
  try {
    await program.parseAsync(process.argv);
  } catch (err) {
    reportAndExit(err);
  } finally {
    await flushSentry();
  }
};

void main();
