import type { KyInstance } from 'ky';
import { z } from 'zod';
import type { AuthStore } from '../auth/store.ts';
import { createHttpClient } from '../http/client.ts';
import {
  type BottleneckItem,
  bottlenecksResponseSchema,
  type CreateResearchRunResponse,
  citationsResponseSchema,
  createResearchRunResponseSchema,
  type HypothesisItem,
  hypothesesResponseSchema,
  type Me,
  meSchema,
  type Pattern,
  type PersonaItem,
  type PersonaProfile,
  type ProductDetail,
  type ProductListItem,
  personaProfileSchema,
  personasResponseSchema,
  productDetailSchema,
  productListItemSchema,
  type ResearchMode,
  type ResearchRun,
  researchModeSchema,
  researchRunSchema,
  statusResponseSchema,
  type WorkflowStartResponse,
  workflowStartResponseSchema,
} from './types.ts';

export interface ApiClientOptions {
  apiBase: string;
  auth: AuthStore;
}

const parseArray = <T>(schema: z.ZodType<T>, raw: unknown): T[] => z.array(schema).parse(raw);

export class ApiClient {
  private readonly http: KyInstance;

  constructor(opts: ApiClientOptions) {
    this.http = createHttpClient({ apiBase: opts.apiBase, auth: opts.auth });
  }

  async me(): Promise<Me> {
    return meSchema.parse(await this.http.get('me').json());
  }

  async listProducts(): Promise<ProductListItem[]> {
    return parseArray(productListItemSchema, await this.http.get('products').json());
  }

  async getProduct(id: string): Promise<ProductDetail> {
    return productDetailSchema.parse(await this.http.get(`products/${id}`).json());
  }

  async listPersonas(productId: string): Promise<{
    analysisStatus: string;
    data: PersonaItem[];
  }> {
    const parsed = personasResponseSchema.parse(
      await this.http.get(`products/${productId}/personas`).json(),
    );
    return { analysisStatus: parsed.analysis_status, data: parsed.data };
  }

  async getPersonaProfile(productId: string, personaId: string): Promise<PersonaProfile> {
    return personaProfileSchema.parse(
      await this.http.get(`products/${productId}/personas/${personaId}/profile`).json(),
    );
  }

  async getPersonaCitations(
    productId: string,
    personaId: string,
    traitIndex: number,
  ): Promise<unknown> {
    return citationsResponseSchema.parse(
      await this.http
        .get(`products/${productId}/personas/${personaId}/citations`, {
          searchParams: { trait_index: traitIndex },
        })
        .json(),
    );
  }

  async listHypotheses(productId: string): Promise<HypothesisItem[]> {
    const parsed = hypothesesResponseSchema.parse(
      await this.http.get(`products/${productId}/hypotheses`).json(),
    );
    return parsed.data;
  }

  async listBottlenecks(productId: string): Promise<BottleneckItem[]> {
    const parsed = bottlenecksResponseSchema.parse(
      await this.http.get(`products/${productId}/bottlenecks`).json(),
    );
    return parsed.data;
  }

  async listPatterns(productId: string): Promise<Pattern[]> {
    const [hypotheses, bottlenecks] = await Promise.all([
      this.listHypotheses(productId),
      this.listBottlenecks(productId),
    ]);
    return [
      ...hypotheses.map((h): Pattern => ({ ...h, kind: 'hypothesis' })),
      ...bottlenecks.map((b): Pattern => ({ ...b, kind: 'bottleneck' })),
    ];
  }

  async startSynthesis(productId: string): Promise<{ status: string }> {
    return statusResponseSchema.parse(await this.http.post(`products/${productId}/context`).json());
  }

  async startAnalysis(productId: string): Promise<{ status: string }> {
    return statusResponseSchema.parse(await this.http.post(`products/${productId}/enrich`).json());
  }

  async generatePersonas(productId: string, personaCount?: number): Promise<WorkflowStartResponse> {
    return workflowStartResponseSchema.parse(
      await this.http
        .post(`products/${productId}/personas/generate`, {
          json: personaCount === undefined ? {} : { persona_count: personaCount },
        })
        .json(),
    );
  }

  async listResearchModes(): Promise<ResearchMode[]> {
    return parseArray(researchModeSchema, await this.http.get('research-modes').json());
  }

  async listResearchRuns(productId: string): Promise<ResearchRun[]> {
    return parseArray(
      researchRunSchema,
      await this.http.get(`products/${productId}/research-runs`).json(),
    );
  }

  async getResearchRun(productId: string, runId: string): Promise<ResearchRun> {
    return researchRunSchema.parse(
      await this.http.get(`products/${productId}/research-runs/${runId}`).json(),
    );
  }

  async createResearchRun(
    productId: string,
    body: {
      kind: 'interview' | 'ab_test' | 'delphi' | 'playtest';
      title?: string;
      config: Record<string, unknown>;
      participant_rule?: Record<string, unknown>;
    },
  ): Promise<CreateResearchRunResponse> {
    return createResearchRunResponseSchema.parse(
      await this.http.post(`products/${productId}/research-runs`, { json: body }).json(),
    );
  }
}
