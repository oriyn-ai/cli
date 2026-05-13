#!/usr/bin/env bun
import { existsSync, mkdirSync, readdirSync, writeFileSync } from 'node:fs';
import pkg from '../package.json' with { type: 'json' };

const CHANGESET_DIR = '.changeset';
const BASE_REF = process.env.CHANGESET_BASE_REF ?? 'origin/main';
const text = new TextDecoder();

type DependencySection = 'dependencies' | 'optionalDependencies' | 'peerDependencies';

type PackageJson = {
  name?: unknown;
  dependencies?: Record<string, string>;
  devDependencies?: Record<string, string>;
  optionalDependencies?: Record<string, string>;
  peerDependencies?: Record<string, string>;
};

type DependencyChange = {
  name: string;
  from?: string;
  to?: string;
};

const dependencySections: DependencySection[] = [
  'dependencies',
  'optionalDependencies',
  'peerDependencies',
];

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

const hasChangeset = (): boolean => {
  if (!existsSync(CHANGESET_DIR)) return false;

  return readdirSync(CHANGESET_DIR).some((file) => file.endsWith('.md') && file !== 'README.md');
};

const collectDependencyChanges = (
  basePackage: PackageJson,
  headPackage: PackageJson,
): DependencyChange[] => {
  const changes: DependencyChange[] = [];

  for (const section of dependencySections) {
    const baseDependencies = basePackage[section] ?? {};
    const headDependencies = headPackage[section] ?? {};
    const names = new Set([...Object.keys(baseDependencies), ...Object.keys(headDependencies)]);

    for (const name of [...names].sort()) {
      const from = baseDependencies[name];
      const to = headDependencies[name];

      if (from !== to) {
        changes.push({ name, from, to });
      }
    }
  }

  return changes;
};

const formatDependency = ({ name, from, to }: DependencyChange): string => {
  if (from && to) return `${name} from ${from} to ${to}`;
  if (to) return `${name} to ${to}`;
  return name;
};

if (hasChangeset()) {
  console.log('Changeset already exists; no Dependabot changeset needed.');
  process.exit(0);
}

const basePackageRaw = git(['show', `${BASE_REF}:package.json`]);
const basePackage = JSON.parse(basePackageRaw) as PackageJson;
const headPackage = pkg as PackageJson;
const dependencyChanges = collectDependencyChanges(basePackage, headPackage);

if (dependencyChanges.length === 0) {
  console.log('No dependency changes found; no Dependabot changeset needed.');
  process.exit(0);
}

mkdirSync(CHANGESET_DIR, { recursive: true });

const packageName = typeof headPackage.name === 'string' ? headPackage.name : 'oriyn';
const prNumber = process.env.PR_NUMBER;
const filename = `${CHANGESET_DIR}/dependabot-${prNumber ?? 'dependencies'}.md`;
const firstDependencyChange = dependencyChanges[0];
const summary =
  dependencyChanges.length === 1 && firstDependencyChange
    ? `Bump ${formatDependency(firstDependencyChange)}.`
    : `Bump dependencies: ${dependencyChanges.map(formatDependency).join(', ')}.`;

writeFileSync(filename, `---\n"${packageName}": patch\n---\n\n${summary}\n`);

console.log(`Wrote ${filename}.`);
