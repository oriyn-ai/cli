import { z } from 'zod';
import { readJson, writeJsonAtomic } from '../storage/file.ts';

export const projectLinkSchema = z.object({
  $schema: z.string().optional(),
  orgId: z.string(),
  productId: z.string(),
});

export type ProjectLink = z.infer<typeof projectLinkSchema>;

const SCHEMA_URL = 'https://oriyn.ai/schema/oriyn.json';

export const readProjectLink = async (path: string): Promise<ProjectLink | null> => {
  const raw = await readJson<unknown>(path);
  if (!raw) return null;
  const parsed = projectLinkSchema.safeParse(raw);
  if (!parsed.success) return null;
  return parsed.data;
};

export const writeProjectLink = async (
  path: string,
  link: Pick<ProjectLink, 'orgId' | 'productId'>,
): Promise<void> => {
  const payload: ProjectLink = { $schema: SCHEMA_URL, ...link };
  await writeJsonAtomic(path, payload, { mode: 0o644 });
};
