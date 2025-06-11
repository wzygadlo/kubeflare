package ratelimit

import (
	"context"
	"errors"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/replicatedhq/kubeflare/pkg/apis/kubeflare/v1alpha1"
)

// Client handles Cloudflare rate limit operations
type Client struct {
	api *cf.API
}

// NewClient creates a new rate limit client
func NewClient(api *cf.API) *Client {
	return &Client{
		api: api,
	}
}

// Create creates a new rate limit in Cloudflare
func (c *Client) Create(ctx context.Context, rateLimit *v1alpha1.RateLimit) (string, error) {
	if c.api == nil {
		return "", errors.New("cloudflare API client is not initialized")
	}

	// Convert to Cloudflare rate limit model
	cfRateLimit := convertToCFRateLimit(rateLimit)

	// Call Cloudflare API
	result, err := c.api.CreateRateLimit(ctx, rateLimit.Spec.ZoneID, cfRateLimit)
	if err != nil {
		return "", err
	}

	return result.ID, nil
}

// Get retrieves a rate limit from Cloudflare
func (c *Client) Get(ctx context.Context, zoneID, rateLimitID string) (*cf.RateLimit, error) {
	if c.api == nil {
		return nil, errors.New("cloudflare API client is not initialized")
	}

	return c.api.RateLimit(ctx, zoneID, rateLimitID)
}

// Update updates an existing rate limit in Cloudflare
func (c *Client) Update(ctx context.Context, rateLimit *v1alpha1.RateLimit) error {
	if c.api == nil {
		return errors.New("cloudflare API client is not initialized")
	}

	if rateLimit.Status.ID == "" {
		return errors.New("rate limit ID is missing")
	}

	// Convert to Cloudflare rate limit model
	cfRateLimit := convertToCFRateLimit(rateLimit)

	// Call Cloudflare API
	_, err := c.api.UpdateRateLimit(ctx, rateLimit.Spec.ZoneID, rateLimit.Status.ID, cfRateLimit)
	return err
}

// Delete deletes a rate limit from Cloudflare
func (c *Client) Delete(ctx context.Context, zoneID, rateLimitID string) error {
	if c.api == nil {
		return errors.New("cloudflare API client is not initialized")
	}

	return c.api.DeleteRateLimit(ctx, zoneID, rateLimitID)
}

// List lists all rate limits for a zone
func (c *Client) List(ctx context.Context, zoneID string) ([]cf.RateLimit, error) {
	if c.api == nil {
		return nil, errors.New("cloudflare API client is not initialized")
	}

	return c.api.ListRateLimits(ctx, zoneID)
}

// convertToCFRateLimit converts from the CRD model to Cloudflare API model
func convertToCFRateLimit(rateLimit *v1alpha1.RateLimit) cf.RateLimit {
	cfRateLimit := cf.RateLimit{
		Description: rateLimit.Spec.Description,
		Threshold:   rateLimit.Spec.Threshold,
		Period:      rateLimit.Spec.Period,
		Disabled:    rateLimit.Spec.Disabled,
	}

	// Convert match criteria
	cfMatch := cf.RateLimitMatch{}
	if len(rateLimit.Spec.Match.Methods) > 0 {
		cfMatch.Request.Methods = rateLimit.Spec.Match.Methods
	}
	if len(rateLimit.Spec.Match.Schemes) > 0 {
		cfMatch.Request.Schemes = rateLimit.Spec.Match.Schemes
	}
	if len(rateLimit.Spec.Match.URL.Patterns) > 0 {
		cfMatch.Request.URLPattern = rateLimit.Spec.Match.URL.Patterns
	}
	cfRateLimit.Match = cfMatch

	// Convert action
	cfAction := cf.RateLimitAction{
		Mode:    rateLimit.Spec.Action.Mode,
		Timeout: rateLimit.Spec.Action.Timeout,
	}
	if rateLimit.Spec.Action.Response != nil {
		cfAction.Response = &cf.RateLimitActionResponse{
			ContentType: rateLimit.Spec.Action.Response.ContentType,
			Body:        rateLimit.Spec.Action.Response.Body,
		}
	}
	cfRateLimit.Action = cfAction

	return cfRateLimit
}

// ConvertFromCF converts from Cloudflare API model to CRD model
func ConvertFromCF(cfRateLimit *cf.RateLimit) *v1alpha1.RateLimitSpec {
	spec := &v1alpha1.RateLimitSpec{
		Description: cfRateLimit.Description,
		Threshold:   cfRateLimit.Threshold,
		Period:      cfRateLimit.Period,
		Disabled:    cfRateLimit.Disabled,
	}

	// Convert match criteria
	match := v1alpha1.RateLimitMatch{}
	if cfRateLimit.Match.Request.Methods != nil {
		match.Methods = cfRateLimit.Match.Request.Methods
	}
	if cfRateLimit.Match.Request.Schemes != nil {
		match.Schemes = cfRateLimit.Match.Request.Schemes
	}
	if cfRateLimit.Match.Request.URLPattern != nil {
		match.URL.Patterns = cfRateLimit.Match.Request.URLPattern
	}
	spec.Match = match

	// Convert action
	action := v1alpha1.RateLimitAction{
		Mode:    cfRateLimit.Action.Mode,
		Timeout: cfRateLimit.Action.Timeout,
	}
	if cfRateLimit.Action.Response != nil {
		action.Response = &v1alpha1.RateLimitActionResponse{
			ContentType: cfRateLimit.Action.Response.ContentType,
			Body:        cfRateLimit.Action.Response.Body,
		}
	}
	spec.Action = action

	return spec
}
