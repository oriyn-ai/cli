#!/usr/bin/env bun
import { spawn } from 'node:child_process';
import { mkdir, rm } from 'node:fs/promises';
import { join } from 'node:path';

interface Target {
  triple: string;
  outName: string;
}

const TARGETS: Target[] = [
  { triple: 'bun-darwin-arm64', outName: 'oriyn-darwin-arm64' },
  { triple: 'bun-darwin-x64', outName: 'oriyn-darwin-x64' },
  { triple: 'bun-linux-x64', outName: 'oriyn-linux-x64' },
  { triple: 'bun-linux-arm64', outName: 'oriyn-linux-arm64' },
  { triple: 'bun-windows-x64', outName: 'oriyn-windows-x64.exe' },
];

const VERSION = process.env.ORIYN_VERSION ?? '0.0.0-dev';
const COMMIT = process.env.ORIYN_COMMIT ?? 'unknown';
const ROOT = new URL('..', import.meta.url).pathname;
const OUT_DIR = join(ROOT, 'dist', 'bin');

const run = (cmd: string, args: string[]): Promise<void> =>
  new Promise((res, rej) => {
    const child = spawn(cmd, args, { stdio: 'inherit', cwd: ROOT });
    child.on('close', (code) => (code === 0 ? res() : rej(new Error(`exit ${code}`))));
    child.on('error', rej);
  });

await rm(OUT_DIR, { recursive: true, force: true });
await mkdir(OUT_DIR, { recursive: true });

for (const target of TARGETS) {
  const outFile = join(OUT_DIR, target.outName);
  console.log(`-> ${target.triple}`);
  await run('bun', [
    'build',
    'src/index.ts',
    '--compile',
    `--target=${target.triple}`,
    '--define',
    `__ORIYN_VERSION__='${VERSION}'`,
    '--define',
    `__ORIYN_COMMIT__='${COMMIT}'`,
    '--outfile',
    outFile,
  ]);
}

console.log(`\nBuilt ${TARGETS.length} binaries → ${OUT_DIR}`);
