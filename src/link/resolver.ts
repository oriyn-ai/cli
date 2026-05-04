import { homedir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { readEnv } from '../env.ts';
import { PROJECT_LINK_FILENAME } from '../storage/paths.ts';
import { type ProjectLink, readProjectLink } from './file.ts';

export interface ResolvedProduct {
  productId: string;
  orgId: string | undefined;
  source: 'flag' | 'env' | 'oriyn.json' | 'config';
  /** Path to the oriyn.json that was used, when source === 'oriyn.json'. */
  linkPath?: string;
}

export interface ResolveOptions {
  flagProduct?: string;
  flagOrg?: string;
  cwd?: string;
  configFallback?: { productId: string; orgId?: string };
  env?: NodeJS.ProcessEnv;
}

export const findProjectLinkPath = async (cwd: string = process.cwd()): Promise<string | null> => {
  const home = homedir();
  let dir = resolve(cwd);
  while (true) {
    const candidate = join(dir, PROJECT_LINK_FILENAME);
    if (await Bun.file(candidate).exists()) return candidate;
    const parent = dirname(dir);
    if (parent === dir) return null;
    if (dir === home) return null;
    dir = parent;
  }
};

export const resolveProduct = async (
  opts: ResolveOptions = {},
): Promise<ResolvedProduct | null> => {
  if (opts.flagProduct) {
    return {
      productId: opts.flagProduct,
      orgId: opts.flagOrg,
      source: 'flag',
    };
  }
  const env = readEnv(opts.env);
  if (env.product) {
    return {
      productId: env.product,
      orgId: env.org,
      source: 'env',
    };
  }
  const linkPath = await findProjectLinkPath(opts.cwd);
  if (linkPath) {
    const link: ProjectLink | null = await readProjectLink(linkPath);
    if (link) {
      return {
        productId: link.productId,
        orgId: link.orgId,
        source: 'oriyn.json',
        linkPath,
      };
    }
  }
  if (opts.configFallback) {
    return {
      productId: opts.configFallback.productId,
      orgId: opts.configFallback.orgId,
      source: 'config',
    };
  }
  return null;
};
