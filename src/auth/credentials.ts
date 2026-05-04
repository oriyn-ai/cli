import { z } from 'zod';

export const CREDENTIALS_SCHEMA_VERSION = 1 as const;

export const credentialsSchema = z.object({
  schema_version: z.literal(CREDENTIALS_SCHEMA_VERSION),
  access_token: z.string().min(1),
  refresh_token: z.string().min(1),
  expires_at: z.number().int().nonnegative(),
});

export type StoredCredentials = z.infer<typeof credentialsSchema>;
