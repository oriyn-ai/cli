#!/usr/bin/env bun
import { appendFileSync, existsSync, readdirSync } from 'node:fs';
import pkg from '../package.json' with { type: 'json' };

const CHANGESET_DIR = '.changeset';
const checkOnly = process.argv.includes('--check');
const dryRun = process.argv.includes('--dry-run') || process.env.DRY_RUN === '1';
const text = new TextDecoder();

const git = (args: string[], allowFailure = false): string => {
  const result = Bun.spawnSync(['git', ...args], {
    stdout: 'pipe',
    stderr: 'pipe',
  });

  const stdout = text.decode(result.stdout).trim();
  const stderr = text.decode(result.stderr).trim();

  if (result.exitCode !== 0 && !allowFailure) {
    throw new Error(`git ${args.join(' ')} failed${stderr ? `: ${stderr}` : ''}`);
  }

  return stdout;
};

const hasPendingChangesets = (): boolean => {
  if (!existsSync(CHANGESET_DIR)) return false;

  return readdirSync(CHANGESET_DIR).some((file) => file.endsWith('.md') && file !== 'README.md');
};

const setOutput = (name: string, value: string): void => {
  if (!process.env.GITHUB_OUTPUT) return;
  appendFileSync(process.env.GITHUB_OUTPUT, `${name}=${value}\n`);
};

const finish = (needed: boolean, message: string): never => {
  setOutput('needed', String(needed));
  setOutput('tag', tag);
  setOutput('version', String(pkg.version));
  console.log(message);
  process.exit(0);
};

const tag = `v${pkg.version}`;

if (hasPendingChangesets()) {
  finish(false, 'Pending changesets found; skipping release tag creation.');
}

if (!dryRun) {
  git(['fetch', '--tags', 'origin']);
}

const tagRef = `refs/tags/${tag}`;
const existingTag = git(['rev-parse', '--verify', tagRef], true);

if (existingTag) {
  finish(false, `Release tag ${tag} already exists; no tag needed.`);
}

const sha = git(['rev-parse', '--short', 'HEAD']);

if (checkOnly || dryRun) {
  finish(true, `Release tag ${tag} is needed at ${sha}.`);
}

git(['tag', tag]);
git(['push', 'origin', tag]);

console.log(`Created and pushed release tag ${tag} at ${sha}.`);
