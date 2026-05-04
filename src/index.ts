#!/usr/bin/env bun
import { Command } from 'commander';
import { VERSION } from './version.ts';

const program = new Command();

program
  .name('oriyn')
  .description('Predict how users respond to product changes before shipping.')
  .version(VERSION, '-v, --version', 'output the version number')
  .helpOption('-h, --help', 'display help for command')
  .showHelpAfterError();

await program.parseAsync(process.argv);
