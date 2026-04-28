package apiclient

import (
	"encoding/json"
	"fmt"
)

type MeResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type ProductContext struct {
	Company          string   `json:"company"`
	ProductSummary   string   `json:"product_summary"`
	CoreFeatures     []string `json:"core_features"`
	TargetUsers      string   `json:"target_users"`
	ValueProposition string   `json:"value_proposition"`
	UseCases         []string `json:"use_cases"`
}

type ProductListItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextStatus string `json:"context_status"`
}

type ProductDetail struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Context          *ProductContext `json:"context"`
	ContextStatus    string          `json:"context_status"`
	EnrichmentStatus string          `json:"enrichment_status"`
	CreatedAt        string          `json:"created_at"`
}

type UpdateContextRequest struct {
	Company          *string   `json:"company,omitempty"`
	ProductSummary   *string   `json:"product_summary,omitempty"`
	CoreFeatures     *[]string `json:"core_features,omitempty"`
	TargetUsers      *string   `json:"target_users,omitempty"`
	ValueProposition *string   `json:"value_proposition,omitempty"`
	UseCases         *[]string `json:"use_cases,omitempty"`
}

type ContextVersion struct {
	ID        string         `json:"id"`
	Version   int            `json:"version"`
	Context   ProductContext `json:"context"`
	Source    string         `json:"source"`
	CreatedAt string         `json:"created_at"`
}

type ContextHistoryResponse struct {
	Versions []ContextVersion `json:"versions"`
}

type EnrichmentData[T any] struct {
	EnrichmentStatus string `json:"enrichment_status"`
	Data             []T    `json:"data"`
}

type PersonaItem struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	BehavioralTraits    []string `json:"behavioral_traits"`
	SizeEstimate        int      `json:"size_estimate"`
	GeneratedAt         string   `json:"generated_at"`
	Status              string   `json:"status"`
	UpdatedAt           *string  `json:"updated_at"`
	TraitCitationCounts []int    `json:"trait_citation_counts"`
}

type HypothesisItem struct {
	Sequence         []string `json:"sequence"`
	RenderedSequence []string `json:"rendered_sequence"`
	Frequency        int      `json:"frequency"`
	UserCount        int      `json:"user_count"`
	SignificancePct  float64  `json:"significance_pct"`
	SourceUsers      []string `json:"source_users"`
}

type BottleneckItem struct {
	Sequence           []string `json:"sequence"`
	RenderedSequence   []string `json:"rendered_sequence"`
	Traversals         int      `json:"traversals"`
	UserCount          int      `json:"user_count"`
	AvgDurationSeconds float64  `json:"avg_duration_seconds"`
	SourceUsers        []string `json:"source_users"`
}

type CitationItem struct {
	ID                string  `json:"id"`
	SessionAssetID    string  `json:"session_asset_id"`
	ExternalSessionID string  `json:"external_session_id"`
	SessionSummary    string  `json:"session_summary"`
	FrustrationScore  float64 `json:"frustration_score"`
	DurationMS        int     `json:"duration_ms"`
	RecordedAt        string  `json:"recorded_at"`
	ReplayURL         *string `json:"replay_url"`
	HasStoredReplay   bool    `json:"has_stored_replay"`
}

type CitationsResponse struct {
	Citations []CitationItem `json:"citations"`
}

type PersonaProfileResponse struct {
	StaticFacts  []string `json:"static_facts"`
	DynamicFacts []string `json:"dynamic_facts"`
}

type KnowledgeSearchRequest struct {
	Query     string  `json:"query"`
	Limit     int     `json:"limit,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
	Rerank    bool    `json:"rerank,omitempty"`
}

type KnowledgeSearchResult struct {
	Content   string                 `json:"content"`
	Score     float64                `json:"score"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt string                 `json:"created_at"`
}

type KnowledgeSearchResponse struct {
	Results []KnowledgeSearchResult `json:"results"`
}

type TimelineItem struct {
	Kind             string                 `json:"kind"`
	Provider         string                 `json:"provider"`
	ExternalID       string                 `json:"external_id"`
	EventName        string                 `json:"event_name"`
	Timestamp        string                 `json:"timestamp"`
	Properties       map[string]interface{} `json:"properties"`
	SessionAssetID   *string                `json:"session_asset_id"`
	DurationMS       *int                   `json:"duration_ms"`
	SessionSummary   *string                `json:"session_summary"`
	FrustrationScore *float64               `json:"frustration_score"`
	HasStoredReplay  bool                   `json:"has_stored_replay"`
}

type TimelineResponse struct {
	Items []TimelineItem `json:"items"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type CreateExperimentRequest struct {
	Hypothesis string `json:"hypothesis"`
	AgentCount *int   `json:"agent_count,omitempty"`
}

type CreateExperimentResponse struct {
	ExperimentID string `json:"experiment_id"`
}

type PersonaBreakdownItem struct {
	Persona      string  `json:"persona"`
	Response     string  `json:"response"`
	Reasoning    string  `json:"reasoning"`
	AdoptionRate float64 `json:"adoption_rate"`
}

type ExperimentSummary struct {
	Verdict          string                 `json:"verdict"`
	Convergence      float64                `json:"convergence"`
	Summary          string                 `json:"summary"`
	PersonaBreakdown []PersonaBreakdownItem `json:"persona_breakdown"`
	QuestionResults  json.RawMessage        `json:"question_results"`
	AgentCount       int                    `json:"agent_count"`
}

type ExperimentResponse struct {
	ID             string             `json:"id"`
	ProductID      string             `json:"product_id"`
	Hypothesis     string             `json:"hypothesis"`
	Status         string             `json:"status"`
	CreatedByEmail string             `json:"created_by_email"`
	CreatedAt      string             `json:"created_at"`
	Summary        *ExperimentSummary `json:"summary"`
}

type ExperimentListItem struct {
	ID             string   `json:"id"`
	Title          *string  `json:"title"`
	Hypothesis     string   `json:"hypothesis"`
	Status         string   `json:"status"`
	Verdict        *string  `json:"verdict"`
	Convergence    *float64 `json:"convergence"`
	CreatedByEmail string   `json:"created_by_email"`
	CreatedAt      string   `json:"created_at"`
}

type ReplayEventsResponse struct {
	Events []map[string]interface{} `json:"events"`
}

// APIError is returned when the API responds with a structured error body.
// The Oriyn API always includes an "error" field; credits/agent-count errors
// carry additional fields that agents can read to self-correct.
type APIError struct {
	StatusCode       int    `json:"-"`
	Message          string `json:"error"`
	CreditsRequired  *int   `json:"credits_required,omitempty"`
	CreditsAvailable *int   `json:"credits_available,omitempty"`
	MaxAgentCount    *int   `json:"max_agent_count,omitempty"`
	Plan             string `json:"plan,omitempty"`
	Raw              string `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return e.Raw
	}
	return e.Message
}

// PermissionError represents a 403 response from a gated Oriyn API route
// where the server included a permission payload. The CLI renders this as a
// friendly message instead of a raw HTTP error.
type PermissionError struct {
	StatusCode         int
	RequiredPermission string
	Role               string
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf(
		"this action requires `%s`. Your role is `%s`. Ask an Admin for access, or request a role change.",
		e.RequiredPermission, e.Role,
	)
}
