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
  enrichment_status: z.string(),
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
});
export type PersonaItem = z.infer<typeof personaItemSchema>;

export const personasResponseSchema = z.object({
  enrichment_status: z.string(),
  data: z.array(personaItemSchema),
});

export const personaProfileSchema = z.object({
  static_facts: z.array(z.string()),
  dynamic_facts: z.array(z.string()),
});
export type PersonaProfile = z.infer<typeof personaProfileSchema>;

export const citationItemSchema = z.object({
  id: z.string(),
  session_asset_id: z.string(),
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
  source_users: z.array(z.string()),
});
export type HypothesisItem = z.infer<typeof hypothesisItemSchema>;

export const hypothesesResponseSchema = z.object({
  enrichment_status: z.string(),
  data: z.array(hypothesisItemSchema),
});

export const bottleneckItemSchema = z.object({
  sequence: z.array(z.string()),
  rendered_sequence: z.array(z.string()),
  traversals: z.number(),
  user_count: z.number(),
  avg_duration_seconds: z.number(),
  source_users: z.array(z.string()),
});
export type BottleneckItem = z.infer<typeof bottleneckItemSchema>;

export const bottlenecksResponseSchema = z.object({
  enrichment_status: z.string(),
  data: z.array(bottleneckItemSchema),
});

export const personaBreakdownItemSchema = z.object({
  persona: z.string(),
  response: z.string(),
  reasoning: z.string(),
  adoption_rate: z.number(),
});

export const experimentSummarySchema = z.object({
  verdict: z.string(),
  convergence: z.number(),
  summary: z.string(),
  persona_breakdown: z.array(personaBreakdownItemSchema),
  question_results: z.unknown().optional(),
  agent_count: z.number(),
});
export type ExperimentSummary = z.infer<typeof experimentSummarySchema>;

export const experimentResponseSchema = z.object({
  id: z.string(),
  product_id: z.string(),
  hypothesis: z.string(),
  status: z.string(),
  created_by_email: z.string(),
  created_at: z.string(),
  summary: experimentSummarySchema.nullable().optional(),
});
export type ExperimentResponse = z.infer<typeof experimentResponseSchema>;

export const experimentListItemSchema = z.object({
  id: z.string(),
  title: z.string().nullable().optional(),
  hypothesis: z.string(),
  status: z.string(),
  verdict: z.string().nullable().optional(),
  convergence: z.number().nullable().optional(),
  created_by_email: z.string(),
  created_at: z.string(),
});
export type ExperimentListItem = z.infer<typeof experimentListItemSchema>;

export const createExperimentResponseSchema = z.object({
  experiment_id: z.string(),
});

export const statusResponseSchema = z.object({
  status: z.string(),
});

// Unified pattern shape for `oriyn patterns` — discriminator built client-side
// from the two backend endpoints (hypotheses + bottlenecks).
export type Pattern =
  | (HypothesisItem & { kind: 'hypothesis' })
  | (BottleneckItem & { kind: 'bottleneck' });
