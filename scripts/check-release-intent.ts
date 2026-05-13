#!/usr/bin/env bun
import { existsSync } from 'node:fs';
import pkg from '../package.json' with { type: 'json' };

const BASE_REF = process.env.CHANGESET_BASE_REF ?? 'origin/main';
const text = new TextDecoder();
const nonReleasePackageFields = new Set(['devDependencies', 'scripts']);
const releaseRelevantScripts = new Set(['scripts/build-binaries.ts']);

type JsonValue = JsonObject | JsonValue[] | boolean | number | string | null;
type JsonObject = { [key: string]: JsonValue };

const git = (args: string[]): string => {
  const result = Bun.spawnSync(['git', ...args], {
    stdout: 'pipe',
    stderr: 'pipe',
  });

  if (result.exitCode !== 0) {
    const stderr = text.decode(result.stderr).trim();
    throw new Error(`git ${args.join(' ')} failed${stderr ? `: ${stderr}` : ''}`);
  }

  return text.decode(result.stdout).trim();
};

const parseFiles = (output: string): string[] =>
  output
    .split('\n')
    .map((file) => file.trim())
    .filter(Boolean);

const normalizeJson = (value: JsonValue): JsonValue => {
  if (Array.isArray(value)) return value.map(normalizeJson);
  if (!value || typeof value !== 'object') return value;

  return Object.fromEntries(
    Object.entries(value)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, nestedValue]) => [key, normalizeJson(nestedValue)]),
  );
};

const stripNonReleasePackageFields = (packageJson: JsonObject): JsonObject =>
  Object.fromEntries(
    Object.entries(packageJson).filter(([key]) => !nonReleasePackageFields.has(key)),
  );

const hasReleasePackageJsonChanges = (basePackage: JsonObject, headPackage: JsonObject): boolean =>
  JSON.stringify(normalizeJson(stripNonReleasePackageFields(basePackage))) !==
  JSON.stringify(normalizeJson(stripNonReleasePackageFields(headPackage)));

const changedFiles = [
  ...new Set([
    ...parseFiles(git(['diff', '--name-only', `${BASE_REF}...HEAD`])),
    ...parseFiles(git(['diff', '--name-only', '--cached'])),
    ...parseFiles(git(['diff', '--name-only'])),
  ]),
];

const basePackageRaw = git(['show', `${BASE_REF}:package.json`]);
const basePackage = JSON.parse(basePackageRaw) as JsonObject;
const headPackage = pkg as JsonObject;
const packageJsonChanged = changedFiles.includes('package.json');
const packageReleaseChanged =
  packageJsonChanged && hasReleasePackageJsonChanges(basePackage, headPackage);
const packageNonReleaseOnlyChanged = packageJsonChanged && !packageReleaseChanged;

const isReleaseRelevant = (file: string): boolean => {
  if (file === 'package.json') return packageReleaseChanged;
  if (file === 'bun.lock') return !packageNonReleaseOnlyChanged;
  if (file === 'install.sh') return true;

  return file.startsWith('src/') || releaseRelevantScripts.has(file);
};

const relevantChanges = changedFiles.filter(isReleaseRelevant);

if (relevantChanges.length === 0) {
  console.log('No CLI package/runtime changes need release intent.');
  process.exit(0);
}

const hasChangeset = changedFiles.some(
  (file) =>
    file.startsWith('.changeset/') &&
    file.endsWith('.md') &&
    !file.endsWith('README.md') &&
    existsSync(file),
);

const versionBumped =
  typeof basePackage.version === 'string' && basePackage.version !== pkg.version;
const changelogChanged = changedFiles.includes('CHANGELOG.md');

if (hasChangeset || (versionBumped && changelogChanged)) {
  console.log('CLI release intent found.');
  process.exit(0);
}

console.error('CLI package/runtime changes require release intent.');
console.error('');
console.error('Add one of:');
console.error('- a .changeset/*.md file describing the patch/minor/major impact, or');
console.error('- a package.json version bump plus CHANGELOG.md entry for a release/version PR.');
console.error('');
console.error('Release-relevant files:');
for (const file of relevantChanges) console.error(`- ${file}`);
process.exit(1);
