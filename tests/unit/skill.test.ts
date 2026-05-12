import { afterEach, beforeEach, describe, expect, test } from 'bun:test';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import {
  defaultSkillTargets,
  fetchSkillSource,
  installSkillContent,
  parseAgentSelection,
  resolveSkillTargets,
  validateSkillContent,
} from '../../src/commands/skill.ts';

const skill = `---
name: oriyn
description: Validate product decisions against grounded Oriyn personas. Use when a product decision comes up.
metadata:
  version: "3"
---

# Oriyn

Run \`oriyn experiments run "..."\`.
`;

let tmpRoot: string;

beforeEach(async () => {
  tmpRoot = await mkdtemp(join(tmpdir(), 'oriyn-skill-'));
});

afterEach(async () => {
  await rm(tmpRoot, { recursive: true, force: true });
});

describe('skill command helpers', () => {
  test('parseAgentSelection accepts all supported targets', () => {
    expect(parseAgentSelection(undefined)).toBe('all');
    expect(parseAgentSelection('all')).toBe('all');
    expect(parseAgentSelection('Claude')).toBe('claude');
    expect(parseAgentSelection('codex')).toBe('codex');
  });

  test('parseAgentSelection rejects unknown targets', () => {
    expect(() => parseAgentSelection('cursor')).toThrow('--agent must be one of');
  });

  test('defaultSkillTargets returns Claude Code and Codex locations', () => {
    const targets = defaultSkillTargets(tmpRoot);
    expect(targets.claude.file).toBe(join(tmpRoot, '.claude', 'skills', 'oriyn', 'SKILL.md'));
    expect(targets.codex.file).toBe(join(tmpRoot, '.agents', 'skills', 'oriyn', 'SKILL.md'));
  });

  test('resolveSkillTargets defaults to both agents', () => {
    const targets = resolveSkillTargets({ home: tmpRoot });
    expect(targets.map((target) => target.agent)).toEqual(['claude', 'codex']);
  });

  test('resolveSkillTargets supports a custom path for one agent', () => {
    const custom = join(tmpRoot, 'custom-skill');
    const targets = resolveSkillTargets({ agent: 'codex', path: custom });
    expect(targets).toHaveLength(1);
    expect(targets[0]?.agent).toBe('codex');
    expect(targets[0]?.file).toBe(join(custom, 'SKILL.md'));
  });

  test('resolveSkillTargets rejects custom path with all agents', () => {
    expect(() => resolveSkillTargets({ agent: 'all', path: tmpRoot })).toThrow(
      '--path can only be used',
    );
  });

  test('validateSkillContent requires oriyn name and description', () => {
    expect(() => validateSkillContent(skill)).not.toThrow();
    expect(() => validateSkillContent(skill.replace('name: oriyn', 'name: other'))).toThrow(
      'name: oriyn',
    );
    expect(() => validateSkillContent(skill.replace('description:', 'summary:'))).toThrow(
      'description',
    );
  });

  test('fetchSkillSource reads local skill files', async () => {
    const path = join(tmpRoot, 'SKILL.md');
    await writeFile(path, skill);
    await expect(fetchSkillSource(path)).resolves.toEqual({ content: skill, source: path });
  });

  test('fetchSkillSource reads HTTP skill sources', async () => {
    const originalFetch = globalThis.fetch;
    const source = 'https://oriyn.ai/skill.md';
    globalThis.fetch = (async () => new Response(skill)) as unknown as typeof fetch;

    try {
      await expect(fetchSkillSource(source)).resolves.toEqual({ content: skill, source });
    } finally {
      globalThis.fetch = originalFetch;
    }
  });

  test('installSkillContent writes each target and refuses accidental overwrite', async () => {
    const targets = resolveSkillTargets({ home: tmpRoot });
    const results = await installSkillContent({ content: skill, targets });

    expect(results.map((result) => result.existed)).toEqual([false, false]);
    for (const target of targets) {
      expect(await readFile(target.file, 'utf8')).toBe(skill);
    }

    await expect(installSkillContent({ content: skill, targets })).rejects.toThrow(
      'already exists',
    );
  });

  test('installSkillContent overwrites with force', async () => {
    const targetDir = join(tmpRoot, '.agents', 'skills', 'oriyn');
    const targetFile = join(targetDir, 'SKILL.md');
    await mkdir(targetDir, { recursive: true });
    await writeFile(targetFile, 'old');

    const targets = resolveSkillTargets({ agent: 'codex', home: tmpRoot });
    const results = await installSkillContent({ content: skill, force: true, targets });

    expect(results).toHaveLength(1);
    expect(results[0]?.existed).toBe(true);
    expect(await readFile(targetFile, 'utf8')).toBe(skill);
  });
});
