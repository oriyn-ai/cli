import type { KyInstance } from 'ky';
import { z } from 'zod';
import type { AuthStore } from '../auth/store.ts';
import { createHttpClient } from '../http/client.ts';
import {
  type BottleneckItem,
  bottlenecksResponseSchema,
  citationsResponseSchema,
  createExperimentResponseSchema,
  type ExperimentListItem,
  type ExperimentResponse,
  experimentListItemSchema,
  experimentResponseSchema,
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
  statusResponseSchema,
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

  async listExperiments(productId: string): Promise<ExperimentListItem[]> {
    return parseArray(
      experimentListItemSchema,
      await this.http.get(`products/${productId}/experiments`).json(),
    );
  }

  async getExperiment(productId: string, experimentId: string): Promise<ExperimentResponse> {
    return experimentResponseSchema.parse(
      await this.http.get(`products/${productId}/experiments/${experimentId}`).json(),
    );
  }

  async createExperiment(
    productId: string,
    body: { hypothesis: string; agent_count?: number },
  ): Promise<{ experimentId: string; url: string }> {
    const parsed = createExperimentResponseSchema.parse(
      await this.http.post(`products/${productId}/experiments`, { json: body }).json(),
    );
    return { experimentId: parsed.experiment_id, url: parsed.url };
  }

  async startSynthesis(productId: string): Promise<{ status: string }> {
    return statusResponseSchema.parse(await this.http.post(`products/${productId}/context`).json());
  }

  async startAnalysis(productId: string): Promise<{ status: string }> {
    return statusResponseSchema.parse(await this.http.post(`products/${productId}/enrich`).json());
  }
}
