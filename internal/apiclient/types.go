package apiclient

import "encoding/json"

type MeResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type ProductListItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextStatus string `json:"context_status"`
}

type ProductDetail struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      *string         `json:"description"`
	URLs             []string        `json:"urls"`
	Context          json.RawMessage `json:"context"`
	ContextStatus    string          `json:"context_status"`
	EnrichmentStatus string          `json:"enrichment_status"`
	CreatedAt        string          `json:"created_at"`
}

type EnrichmentData[T any] struct {
	EnrichmentStatus string `json:"enrichment_status"`
	Data             []T    `json:"data"`
}

type PersonaItem struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	BehavioralTraits json.RawMessage `json:"behavioral_traits"`
	SizeEstimate     int             `json:"size_estimate"`
	GeneratedAt      string          `json:"generated_at"`
}

type PatternItem struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Frequency    string          `json:"frequency"`
	Significance string          `json:"significance"`
	RawSequence  json.RawMessage `json:"raw_sequence"`
	GeneratedAt  string          `json:"generated_at"`
}

type RecommendationItem struct {
	Title     string `json:"title"`
	Rationale string `json:"rationale"`
	Priority  string `json:"priority"`
}

type DirectionItem struct {
	ID              string               `json:"id"`
	Recommendations []RecommendationItem `json:"recommendations"`
	DerivedFrom     json.RawMessage      `json:"derived_from"`
	GeneratedAt     string               `json:"generated_at"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type CreateExperimentRequest struct {
	Hypothesis string `json:"hypothesis"`
}

type CreateExperimentResponse struct {
	ExperimentID string `json:"experiment_id"`
}

type ExperimentSummary struct {
	Verdict          string                 `json:"verdict"`
	Confidence       float32                `json:"confidence"`
	Summary          string                 `json:"summary"`
	PersonaBreakdown []PersonaBreakdownItem `json:"persona_breakdown"`
}

type PersonaBreakdownItem struct {
	Persona   string `json:"persona"`
	Response  string `json:"response"`
	Reasoning string `json:"reasoning"`
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
	ID             string  `json:"id"`
	Hypothesis     string  `json:"hypothesis"`
	Status         string  `json:"status"`
	Verdict        *string `json:"verdict"`
	CreatedByEmail string  `json:"created_by_email"`
	CreatedAt      string  `json:"created_at"`
}
