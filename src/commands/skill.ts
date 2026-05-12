import { access, readFile } from 'node:fs/promises';
import { homedir } from 'node:os';
import { join } from 'node:path';
import type { Command } from 'commander';
import writeFileAtomic from 'write-file-atomic';
import { NetworkError } from '../http/errors.ts';
import { reportAndExit } from '../lib/handle-error.ts';
import { ui } from '../output/human.ts';
import { writeJson } from '../output/jsonl.ts';
import { resolveMode } from '../output/mode.ts';
import { ensureDir } from '../storage/file.ts';

const DEFAULT_SKILL_URL = 'https://oriyn.ai/skill.md';
const SKILL_NAME = 'oriyn';

type AgentTarget = 'claude' | 'codex';
type AgentSelection = AgentTarget | 'all';

type SkillInstallTarget = {
  agent: AgentTarget;
  label: string;
  dir: string;
  file: string;
};

type SkillInstallResult = SkillInstallTarget & {
  existed: boolean;
};

type SkillInstallOptions = {
  agent?: string;
  force?: boolean;
  path?: string;
  source: string;
};

type SkillSource = {
  content: string;
  source: string;
};

const isHttpUrl = (value: string): boolean => /^https?:\/\//i.test(value);

const selectionValues: AgentSelection[] = ['all', 'claude', 'codex'];

export const parseAgentSelection = (value: string | undefined): AgentSelection => {
  const normalized = (value ?? 'all').toLowerCase();
  if (selectionValues.includes(normalized as AgentSelection)) return normalized as AgentSelection;
  throw new Error('--agent must be one of: all, claude, codex');
};

export const defaultSkillTargets = (home = homedir()): Record<AgentTarget, SkillInstallTarget> => {
  const claudeDir = join(home, '.claude', 'skills', SKILL_NAME);
  const codexDir = join(home, '.agents', 'skills', SKILL_NAME);
  return {
    claude: {
      agent: 'claude',
      label: 'Claude Code',
      dir: claudeDir,
      file: join(claudeDir, 'SKILL.md'),
    },
    codex: {
      agent: 'codex',
      label: 'Codex',
      dir: codexDir,
      file: join(codexDir, 'SKILL.md'),
    },
  };
};

export const resolveSkillTargets = ({
  agent,
  home,
  path,
}: {
  agent?: string;
  home?: string;
  path?: string;
}): SkillInstallTarget[] => {
  const selection = parseAgentSelection(agent);
  if (path) {
    if (selection === 'all') {
      throw new Error('--path can only be used with --agent claude or --agent codex');
    }
    return [
      {
        agent: selection,
        label: selection === 'claude' ? 'Claude Code' : 'Codex',
        dir: path,
        file: join(path, 'SKILL.md'),
      },
    ];
  }

  const targets = defaultSkillTargets(home);
  return selection === 'all' ? [targets.claude, targets.codex] : [targets[selection]];
};

export const fetchSkillSource = async (source = DEFAULT_SKILL_URL): Promise<SkillSource> => {
  if (!isHttpUrl(source)) {
    return { content: await readFile(source, 'utf8'), source };
  }

  let response: Response;
  try {
    response = await fetch(source, {
      headers: {
        accept: 'text/markdown,text/plain;q=0.9,*/*;q=0.1',
        'user-agent': 'oriyn-cli-skill-installer',
      },
    });
  } catch (err) {
    throw new NetworkError(`Could not fetch ${source}`, { cause: err });
  }

  if (!response.ok) {
    throw new NetworkError(`Could not fetch ${source}: HTTP ${response.status}`);
  }

  return { content: await response.text(), source };
};

export const validateSkillContent = (content: string): void => {
  if (!content.startsWith('---\n')) {
    throw new Error('Skill content must start with YAML frontmatter');
  }

  const end = content.indexOf('\n---', 4);
  if (end === -1) {
    throw new Error('Skill content is missing closing YAML frontmatter marker');
  }

  const frontmatter = content.slice(4, end).trim();
  if (!/^name:\s*oriyn\s*$/m.test(frontmatter)) {
    throw new Error('Skill frontmatter must contain `name: oriyn`');
  }
  if (!/^description:\s*\S/m.test(frontmatter)) {
    throw new Error('Skill frontmatter must contain a non-empty description');
  }
};

export const installSkillContent = async ({
  content,
  force = false,
  targets,
}: {
  content: string;
  force?: boolean;
  targets: SkillInstallTarget[];
}): Promise<SkillInstallResult[]> => {
  validateSkillContent(content);

  const normalized = content.endsWith('\n') ? content : `${content}\n`;
  const results: SkillInstallResult[] = [];

  for (const target of targets) {
    let existed = true;
    try {
      await access(target.file);
    } catch (err) {
      if ((err as NodeJS.ErrnoException).code !== 'ENOENT') throw err;
      existed = false;
    }

    if (existed && !force) {
      throw new Error(`${target.file} already exists. Re-run with --force to overwrite it.`);
    }

    await ensureDir(target.dir, 0o755);
    await writeFileAtomic(target.file, normalized, { mode: 0o644 });
    results.push({ ...target, existed });
  }

  return results;
};

const emitSkillInstallResult = ({
  source,
  results,
}: {
  source: string;
  results: SkillInstallResult[];
}) => {
  const data = {
    source,
    installed: results.map((result) => ({
      agent: result.agent,
      path: result.file,
      overwritten: result.existed,
    })),
  };

  if (resolveMode() === 'jsonl') {
    writeJson({ type: 'result', data });
    return;
  }

  process.stdout.write(`${ui.bold('Oriyn Agent Skill installed')}\n\n`);
  process.stdout.write(`${ui.dim(`source: ${source}`)}\n`);
  for (const result of results) {
    const verb = result.existed ? 'updated' : 'installed';
    process.stdout.write(`${ui.green(ui.check())} ${result.label}: ${verb} ${result.file}\n`);
  }
  process.stdout.write(
    `\n${ui.dim('Restart Claude Code or Codex if the skill does not appear immediately.')}\n`,
  );
};

const runInstall = async (opts: SkillInstallOptions): Promise<void> => {
  const targets = resolveSkillTargets({
    agent: opts.agent,
    path: opts.path,
  });
  const source = await fetchSkillSource(opts.source);
  const results = await installSkillContent({
    content: source.content,
    force: opts.force,
    targets,
  });
  emitSkillInstallResult({ source: source.source, results });
};

export const registerSkill = (program: Command): void => {
  const skill = program.command('skill').description('Install and inspect the Oriyn Agent Skill');

  skill
    .command('install')
    .description('Install the Oriyn Agent Skill for Claude Code and/or Codex')
    .option('--agent <agent>', 'target agent: all, claude, or codex', 'all')
    .option('--url <url-or-path>', 'skill.md URL or local file path', DEFAULT_SKILL_URL)
    .option('--path <directory>', 'custom target skill directory; use with a single --agent')
    .option('--force', 'overwrite an existing installed skill')
    .action(async (opts: { agent: string; url: string; path?: string; force?: boolean }) => {
      try {
        await runInstall({
          agent: opts.agent,
          force: Boolean(opts.force),
          path: opts.path,
          source: opts.url,
        });
      } catch (err) {
        reportAndExit(err);
      }
    });

  skill
    .command('update')
    .description('Refresh the installed Oriyn Agent Skill from the source')
    .option('--agent <agent>', 'target agent: all, claude, or codex', 'all')
    .option('--url <url-or-path>', 'skill.md URL or local file path', DEFAULT_SKILL_URL)
    .option('--path <directory>', 'custom target skill directory; use with a single --agent')
    .action(async (opts: { agent: string; url: string; path?: string }) => {
      try {
        await runInstall({
          agent: opts.agent,
          force: true,
          path: opts.path,
          source: opts.url,
        });
      } catch (err) {
        reportAndExit(err);
      }
    });

  skill
    .command('print')
    .description('Print the Oriyn Agent Skill without installing it')
    .option('--url <url-or-path>', 'skill.md URL or local file path', DEFAULT_SKILL_URL)
    .action(async (opts: { url: string }) => {
      try {
        const source = await fetchSkillSource(opts.url);
        validateSkillContent(source.content);
        process.stdout.write(
          source.content.endsWith('\n') ? source.content : `${source.content}\n`,
        );
      } catch (err) {
        reportAndExit(err);
      }
    });
};

export default registerSkill;
