package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/go-resty/resty/v2"
)

type AuthProvider interface {
	GetValidAccessToken(ctx context.Context) (string, error)
}

type Client struct {
	resty *resty.Client
}

// apiVersion is the prefix every authenticated call routes through. Bump when
// the server introduces /v2 and the CLI is ready to switch.
const apiVersion = "/v1"

func New(apiBase string, auth AuthProvider) *Client {
	r := resty.New().
		SetBaseURL(apiBase+apiVersion).
		SetHeader("Content-Type", "application/json").
		OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			token, err := auth.GetValidAccessToken(req.Context())
			if err != nil {
				return err
			}
			req.SetAuthToken(token)
			return nil
		})

	return &Client{resty: r}
}

func checkResp(resp *resty.Response, err error) error {
	if err != nil {
		return fmt.Errorf("reaching oriyn API: %w", err)
	}
	if !resp.IsError() {
		return nil
	}

	// 403 responses from gated routes carry required_permission + role at
	// the top level (see oriyn_api.errors._oriyn_error_handler). Lift them
	// into a typed PermissionError so the CLI can render a friendly message
	// and exit with a dedicated code.
	body := resp.Body()
	if resp.StatusCode() == 403 && len(body) > 0 {
		var envelope struct {
			Error              string `json:"error"`
			RequiredPermission string `json:"required_permission"`
			Role               string `json:"role"`
		}
		if jsonErr := json.Unmarshal(body, &envelope); jsonErr == nil && envelope.RequiredPermission != "" {
			return &PermissionError{
				StatusCode:         resp.StatusCode(),
				RequiredPermission: envelope.RequiredPermission,
				Role:               envelope.Role,
			}
		}
	}

	apiErr := &APIError{StatusCode: resp.StatusCode(), Raw: resp.String()}
	if len(body) > 0 {
		_ = json.Unmarshal(body, apiErr)
	}
	if apiErr.Message == "" {
		apiErr.Message = fmt.Sprintf("API returned %d: %s", resp.StatusCode(), resp.String())
	}
	return apiErr
}

func (c *Client) GetMe(ctx context.Context) (*MeResponse, error) {
	var result MeResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/me")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListProducts(ctx context.Context) ([]ProductListItem, error) {
	var result []ProductListItem
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/products")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetProduct(ctx context.Context, id string) (*ProductDetail, error) {
	var result ProductDetail
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/products/" + url.PathEscape(id))
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Synthesize(ctx context.Context, productID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Post("/products/" + url.PathEscape(productID) + "/context")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateContext(ctx context.Context, productID string, body UpdateContextRequest) (*ProductContext, error) {
	var result ProductContext
	resp, err := c.resty.R().SetContext(ctx).SetBody(body).SetResult(&result).
		Patch("/products/" + url.PathEscape(productID) + "/context")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListContextVersions(ctx context.Context, productID string) (*ContextHistoryResponse, error) {
	var result ContextHistoryResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/context/versions")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetContextVersion(ctx context.Context, productID, versionID string) (*ContextVersion, error) {
	var result ContextVersion
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/context/versions/" + url.PathEscape(versionID))
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ScrapeSource(ctx context.Context, productID, sourceID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Post("/products/" + url.PathEscape(productID) + "/sources/" + url.PathEscape(sourceID) + "/scrape")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Enrich(ctx context.Context, productID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Post("/products/" + url.PathEscape(productID) + "/enrich")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPersonas(ctx context.Context, productID string) (*EnrichmentData[PersonaItem], error) {
	var result EnrichmentData[PersonaItem]
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/personas")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPersonaProfile(ctx context.Context, productID, personaID string) (*PersonaProfileResponse, error) {
	var result PersonaProfileResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/personas/" + url.PathEscape(personaID) + "/profile")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPersonaCitations(ctx context.Context, productID, personaID string, traitIndex int) (*CitationsResponse, error) {
	var result CitationsResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		SetQueryParam("trait_index", strconv.Itoa(traitIndex)).
		Get("/products/" + url.PathEscape(productID) + "/personas/" + url.PathEscape(personaID) + "/citations")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetHypotheses(ctx context.Context, productID string) (*EnrichmentData[HypothesisItem], error) {
	var result EnrichmentData[HypothesisItem]
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/hypotheses")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) RefreshHypotheses(ctx context.Context, productID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Post("/products/" + url.PathEscape(productID) + "/hypotheses/refresh")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetBottlenecks(ctx context.Context, productID string) (*EnrichmentData[BottleneckItem], error) {
	var result EnrichmentData[BottleneckItem]
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/bottlenecks")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SearchKnowledge(ctx context.Context, productID string, body KnowledgeSearchRequest) (*KnowledgeSearchResponse, error) {
	var result KnowledgeSearchResponse
	resp, err := c.resty.R().SetContext(ctx).SetBody(body).SetResult(&result).
		Post("/products/" + url.PathEscape(productID) + "/knowledge/search")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetUserTimeline(ctx context.Context, productID, resolvedUserID string, limit int) (*TimelineResponse, error) {
	var result TimelineResponse
	req := c.resty.R().SetContext(ctx).SetResult(&result)
	if limit > 0 {
		req = req.SetQueryParam("limit", strconv.Itoa(limit))
	}
	resp, err := req.Get("/products/" + url.PathEscape(productID) + "/users/" + url.PathEscape(resolvedUserID) + "/timeline")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetSessionReplay(ctx context.Context, productID, sessionAssetID string) (*ReplayEventsResponse, error) {
	var result ReplayEventsResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/sessions/" + url.PathEscape(sessionAssetID) + "/replay")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateExperiment(ctx context.Context, productID string, body CreateExperimentRequest) (*CreateExperimentResponse, error) {
	var result CreateExperimentResponse
	resp, err := c.resty.R().SetContext(ctx).
		SetResult(&result).
		SetBody(body).
		Post("/products/" + url.PathEscape(productID) + "/experiments")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetExperiment(ctx context.Context, productID, experimentID string) (*ExperimentResponse, error) {
	var result ExperimentResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/experiments/" + url.PathEscape(experimentID))
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListExperiments(ctx context.Context, productID string) ([]ExperimentListItem, error) {
	var result []ExperimentListItem
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + url.PathEscape(productID) + "/experiments")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ArchiveExperiment(ctx context.Context, productID, experimentID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Patch("/products/" + url.PathEscape(productID) + "/experiments/" + url.PathEscape(experimentID) + "/archive")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}
