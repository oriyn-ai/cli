#!/usr/bin/env bun
import pkg from '../package.json' with { type: 'json' };

const actual = process.argv[2] ?? process.env.GITHUB_REF_NAME;
const expected = `v${pkg.version}`;

if (!actual) {
  console.error(`Missing release tag. Expected ${expected}.`);
  process.exit(1);
}

if (actual !== expected) {
  console.error(`Release tag ${actual} does not match package version ${expected}.`);
  process.exit(1);
}

console.log(`Release tag ${actual} matches package version ${pkg.version}.`);
