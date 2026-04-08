package apiclient

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"
)

// AuthProvider supplies a valid access token for API requests.
type AuthProvider interface {
	GetValidAccessToken(ctx context.Context) (string, error)
}

// Client is a typed wrapper around the Oriyn API.
type Client struct {
	resty *resty.Client
}

func New(apiBase string, auth AuthProvider) *Client {
	r := resty.New().
		SetBaseURL(apiBase).
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
	if resp.IsError() {
		return fmt.Errorf("API returned %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *Client) GetMe(ctx context.Context) (*MeResponse, error) {
	var result MeResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/v1/me")
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
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/products/" + id)
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPersonas(ctx context.Context, productID string) (*EnrichmentData[PersonaItem], error) {
	var result EnrichmentData[PersonaItem]
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/products/" + productID + "/personas")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPatterns(ctx context.Context, productID string) (*EnrichmentData[PatternItem], error) {
	var result EnrichmentData[PatternItem]
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/products/" + productID + "/patterns")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetDirection(ctx context.Context, productID string) (*EnrichmentData[DirectionItem], error) {
	var result EnrichmentData[DirectionItem]
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Get("/products/" + productID + "/direction")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Synthesize(ctx context.Context, productID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Post("/products/" + productID + "/context")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Enrich(ctx context.Context, productID string) (*StatusResponse, error) {
	var result StatusResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).Post("/products/" + productID + "/enrich")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateExperiment(ctx context.Context, productID, hypothesis string) (*CreateExperimentResponse, error) {
	var result CreateExperimentResponse
	resp, err := c.resty.R().SetContext(ctx).
		SetResult(&result).
		SetBody(CreateExperimentRequest{Hypothesis: hypothesis}).
		Post("/products/" + productID + "/experiments")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetExperiment(ctx context.Context, productID, experimentID string) (*ExperimentResponse, error) {
	var result ExperimentResponse
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + productID + "/experiments/" + experimentID)
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListExperiments(ctx context.Context, productID string) ([]ExperimentListItem, error) {
	var result []ExperimentListItem
	resp, err := c.resty.R().SetContext(ctx).SetResult(&result).
		Get("/products/" + productID + "/experiments")
	if err := checkResp(resp, err); err != nil {
		return nil, err
	}
	return result, nil
}
