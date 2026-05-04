import { z } from 'zod';
import { readJson, writeJsonAtomic } from '../storage/file.ts';
import { cliConfigPath } from '../storage/paths.ts';
import { newDeviceId } from './identity.ts';

export const cliConfigSchema = z.object({
  schema_version: z.literal(1).default(1),
  telemetry: z
    .object({
      enabled: z.boolean().nullable().default(null), // null = undecided
      device_id: z.string().nullable().default(null),
      decided_at: z.string().nullable().default(null),
    })
    .default({ enabled: null, device_id: null, decided_at: null }),
  api_base: z.string().nullable().optional(),
  default_product: z.string().nullable().optional(),
});

export type CliConfig = z.infer<typeof cliConfigSchema>;

const blankConfig = (): CliConfig => cliConfigSchema.parse({});

export const loadCliConfig = async (env: NodeJS.ProcessEnv = process.env): Promise<CliConfig> => {
  const path = cliConfigPath(env);
  const raw = await readJson<unknown>(path);
  if (!raw) return blankConfig();
  const parsed = cliConfigSchema.safeParse(raw);
  if (!parsed.success) return blankConfig();
  return parsed.data;
};

export const saveCliConfig = async (
  config: CliConfig,
  env: NodeJS.ProcessEnv = process.env,
): Promise<void> => {
  const path = cliConfigPath(env);
  await writeJsonAtomic(path, config, { mode: 0o644 });
};

export const ensureDeviceId = async (
  env: NodeJS.ProcessEnv = process.env,
): Promise<{ config: CliConfig; created: boolean }> => {
  const config = await loadCliConfig(env);
  if (config.telemetry.device_id) return { config, created: false };
  const next: CliConfig = {
    ...config,
    telemetry: { ...config.telemetry, device_id: newDeviceId() },
  };
  await saveCliConfig(next, env);
  return { config: next, created: true };
};

export const setTelemetry = async (
  enabled: boolean,
  env: NodeJS.ProcessEnv = process.env,
): Promise<CliConfig> => {
  const config = await loadCliConfig(env);
  const next: CliConfig = {
    ...config,
    telemetry: {
      enabled,
      device_id: config.telemetry.device_id ?? newDeviceId(),
      decided_at: new Date().toISOString(),
    },
  };
  await saveCliConfig(next, env);
  return next;
};
