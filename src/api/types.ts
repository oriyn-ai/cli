import { z } from 'zod';

export const meSchema = z.object({
  user_id: z.string(),
  email: z.string(),
});
export type Me = z.infer<typeof meSchema>;

export const productContextSchema = z.object({
  company: z.string(),
  product_summary: z.string(),
  core_features: z.array(z.string()),
  target_users: z.string(),
  value_proposition: z.string(),
  use_cases: z.array(z.string()),
});
export type ProductContext = z.infer<typeof productContextSchema>;

export const productListItemSchema = z.object({
  id: z.string(),
  name: z.string(),
  context_status: z.string(),
});
export type ProductListItem = z.infer<typeof productListItemSchema>;

export const productDetailSchema = z.object({
  id: z.string(),
  name: z.string(),
  context: productContextSchema.nullable(),
  context_status: z.string(),
  analysis_status: z.string(),
  created_at: z.string(),
});
export type ProductDetail = z.infer<typeof productDetailSchema>;

export const personaItemSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  behavioral_traits: z.array(z.string()),
  size_estimate: z.number(),
  generated_at: z.string(),
  status: z.string(),
  updated_at: z.string().nullable().optional(),
  trait_citation_counts: z.array(z.number()).optional(),
  confidence: z.number().optional(),
  assumption_flags: z.array(z.string()).optional(),
});
export type PersonaItem = z.infer<typeof personaItemSchema>;

export const personasResponseSchema = z.object({
  analysis_status: z.string(),
  data: z.array(personaItemSchema),
});

export const personaProfileSchema = z.object({
  static_facts: z.array(z.string()),
  dynamic_facts: z.array(z.string()),
});
export type PersonaProfile = z.infer<typeof personaProfileSchema>;

export const citationItemSchema = z.object({
  id: z.string(),
  replay_session_id: z.string(),
  external_session_id: z.string(),
  session_summary: z.string(),
  frustration_score: z.number(),
  duration_ms: z.number(),
  recorded_at: z.string(),
  replay_url: z.string().nullable().optional(),
  has_stored_replay: z.boolean(),
});
export type CitationItem = z.infer<typeof citationItemSchema>;

export const citationsResponseSchema = z.object({
  citations: z.array(citationItemSchema),
});

export const hypothesisItemSchema = z.object({
  sequence: z.array(z.string()),
  rendered_sequence: z.array(z.string()),
  frequency: z.number(),
  user_count: z.number(),
  significance_pct: z.number(),
  source_user_ids: z.array(z.string()),
});
export type HypothesisItem = z.infer<typeof hypothesisItemSchema>;

export const hypothesesResponseSchema = z.object({
  analysis_status: z.string(),
  data: z.array(hypothesisItemSchema),
});

export const bottleneckItemSchema = z.object({
  sequence: z.array(z.string()),
  rendered_sequence: z.array(z.string()),
  traversals: z.number(),
  user_count: z.number(),
  avg_duration_seconds: z.number(),
  source_user_ids: z.array(z.string()),
});
export type BottleneckItem = z.infer<typeof bottleneckItemSchema>;

export const bottlenecksResponseSchema = z.object({
  analysis_status: z.string(),
  data: z.array(bottleneckItemSchema),
});

export const workflowStartResponseSchema = z.object({
  status: z.string(),
  workflow_id: z.string().optional(),
});
export type WorkflowStartResponse = z.infer<typeof workflowStartResponseSchema>;

export const researchModeSchema = z.object({
  kind: z.enum(['interview', 'ab_test', 'delphi', 'playtest']),
  config_schema: z.record(z.string(), z.unknown()),
  participant_rules: z.record(z.string(), z.unknown()),
  workflow_name: z.string(),
  task_queue: z.string(),
  output_schema: z.record(z.string(), z.unknown()),
  web_creation_form: z.record(z.string(), z.unknown()),
  web_result_renderer: z.record(z.string(), z.unknown()),
  cli_command: z.record(z.string(), z.unknown()),
  eval_suite: z.array(z.string()),
});
export type ResearchMode = z.infer<typeof researchModeSchema>;

export const researchOutputSchema = z.object({
  id: z.string(),
  participant_id: z.string().nullable().optional(),
  kind: z.string(),
  schema_version: z.number(),
  payload: z.record(z.string(), z.unknown()),
  created_at: z.string(),
});
export type ResearchOutput = z.infer<typeof researchOutputSchema>;

export const researchRunSchema = z.object({
  id: z.string(),
  product_id: z.string(),
  kind: z.enum(['interview', 'ab_test', 'delphi', 'playtest']),
  title: z.string(),
  status: z.string(),
  config: z.record(z.string(), z.unknown()),
  output_summary: z.record(z.string(), z.unknown()).nullable().optional(),
  error: z.string().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
  completed_at: z.string().nullable().optional(),
  outputs: z.array(researchOutputSchema).optional(),
});
export type ResearchRun = z.infer<typeof researchRunSchema>;

export const createResearchRunResponseSchema = z.object({
  research_run_id: z.string(),
  url: z.string().url(),
});
export type CreateResearchRunResponse = z.infer<typeof createResearchRunResponseSchema>;

export const statusResponseSchema = z.object({
  status: z.string(),
});

// Unified pattern shape for `oriyn patterns` — discriminator built client-side
// from the two backend endpoints (hypotheses + bottlenecks).
export type Pattern =
  | (HypothesisItem & { kind: 'hypothesis' })
  | (BottleneckItem & { kind: 'bottleneck' });
