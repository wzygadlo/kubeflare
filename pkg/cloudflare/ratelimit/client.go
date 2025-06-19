package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
)

// Client handles Cloudflare rate limit operations using Rulesets API
type Client struct {
	api *cf.API
}

// NewClient creates a new rate limit client
func NewClient(api *cf.API) *Client {
	return &Client{
		api: api,
	}
}

// RateLimitRule represents a rate limiting rule in the new API
type RateLimitRule struct {
	ID          string
	Expression  string
	Action      string
	Description string
	Enabled     bool
	Requests    int
	Period      int
}

// Create creates a new rate limit rule using Rulesets API
func (c *Client) Create(ctx context.Context, rateLimit *v1alpha1.RateLimit) (string, error) {
	if c.api == nil {
		return "", errors.New("cloudflare API client is not initialized")
	}

	// Get or create the rate limiting ruleset for the zone
	rulesetID, err := c.ensureRateLimitingRuleset(ctx, rateLimit.Spec.ZoneID)
	if err != nil {
		return "", fmt.Errorf("failed to ensure rate limiting ruleset: %w", err)
	}

	// Create the rule within the ruleset
	rule := convertToRulesetRule(rateLimit)

	// Update the entire ruleset with the new rule
	ruleset, err := c.api.GetZoneRuleset(ctx, rateLimit.Spec.ZoneID, rulesetID)
	if err != nil {
		return "", fmt.Errorf("failed to get ruleset: %w", err)
	}

	// Add the new rule to the ruleset
	newRule := rule
	newRule.ID = "" // Let Cloudflare generate the ID
	ruleset.Rules = append(ruleset.Rules, newRule)

	resp, err := c.api.UpdateZoneRuleset(ctx, rateLimit.Spec.ZoneID, rulesetID, ruleset.Description, ruleset.Rules)
	if err != nil {
		return "", fmt.Errorf("failed to create rate limiting rule: %w", err)
	}

	return resp.ID, nil
}

// Get retrieves a rate limit rule from Cloudflare
func (c *Client) Get(ctx context.Context, zoneID, ruleID string) (*RateLimitRule, error) {
	if c.api == nil {
		return nil, errors.New("cloudflare API client is not initialized")
	}

	// Find the rate limiting ruleset
	rulesetID, err := c.findRateLimitingRuleset(ctx, zoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to find rate limiting ruleset: %w", err)
	}

	ruleset, err := c.api.GetZoneRuleset(ctx, zoneID, rulesetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ruleset: %w", err)
	}

	// Find the specific rule
	for _, rule := range ruleset.Rules {
		if rule.ID == ruleID {
			return convertFromRulesetRule(&rule), nil
		}
	}

	return nil, fmt.Errorf("rule with ID %s not found", ruleID)
}

// Update updates an existing rate limit rule
func (c *Client) Update(ctx context.Context, rateLimit *v1alpha1.RateLimit) error {
	if c.api == nil {
		return errors.New("cloudflare API client is not initialized")
	}

	if rateLimit.Status.ID == "" {
		return errors.New("rate limit ID is missing")
	}

	// Find the rate limiting ruleset
	rulesetID, err := c.findRateLimitingRuleset(ctx, rateLimit.Spec.ZoneID)
	if err != nil {
		return fmt.Errorf("failed to find rate limiting ruleset: %w", err)
	}

	// Get the current ruleset
	ruleset, err := c.api.GetZoneRuleset(ctx, rateLimit.Spec.ZoneID, rulesetID)
	if err != nil {
		return fmt.Errorf("failed to get ruleset: %w", err)
	}

	// Find and update the specific rule
	updatedRule := convertToRulesetRule(rateLimit)
	updatedRule.ID = rateLimit.Status.ID

	for i, rule := range ruleset.Rules {
		if rule.ID == rateLimit.Status.ID {
			ruleset.Rules[i] = updatedRule
			break
		}
	}

	// Update the entire ruleset
	_, err = c.api.UpdateZoneRuleset(ctx, rateLimit.Spec.ZoneID, rulesetID, ruleset.Description, ruleset.Rules)
	if err != nil {
		return fmt.Errorf("failed to update rate limiting rule: %w", err)
	}

	return nil
}

// Delete deletes a rate limit rule from Cloudflare
func (c *Client) Delete(ctx context.Context, zoneID, ruleID string) error {
	if c.api == nil {
		return errors.New("cloudflare API client is not initialized")
	}

	// Find the rate limiting ruleset
	rulesetID, err := c.findRateLimitingRuleset(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to find rate limiting ruleset: %w", err)
	}

	// Get the current ruleset
	ruleset, err := c.api.GetZoneRuleset(ctx, zoneID, rulesetID)
	if err != nil {
		return fmt.Errorf("failed to get ruleset: %w", err)
	}

	// Filter out the rule to delete
	var filteredRules []cf.RulesetRule
	for _, rule := range ruleset.Rules {
		if rule.ID != ruleID {
			filteredRules = append(filteredRules, rule)
		}
	}

	// Update the ruleset without the deleted rule
	_, err = c.api.UpdateZoneRuleset(ctx, zoneID, rulesetID, ruleset.Description, filteredRules)
	if err != nil {
		return fmt.Errorf("failed to delete rate limiting rule: %w", err)
	}

	return nil
}

// List lists all rate limiting rules for a zone
func (c *Client) List(ctx context.Context, zoneID string) ([]*RateLimitRule, error) {
	if c.api == nil {
		return nil, errors.New("cloudflare API client is not initialized")
	}

	// Find the rate limiting ruleset
	rulesetID, err := c.findRateLimitingRuleset(ctx, zoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to find rate limiting ruleset: %w", err)
	}

	ruleset, err := c.api.GetZoneRuleset(ctx, zoneID, rulesetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ruleset: %w", err)
	}

	var rules []*RateLimitRule
	for _, rule := range ruleset.Rules {
		if rule.Action == "rate_limit" {
			rules = append(rules, convertFromRulesetRule(&rule))
		}
	}

	return rules, nil
}

// ensureRateLimitingRuleset ensures a rate limiting ruleset exists for the zone
func (c *Client) ensureRateLimitingRuleset(ctx context.Context, zoneID string) (string, error) {
	// Try to find existing rate limiting ruleset
	rulesetID, err := c.findRateLimitingRuleset(ctx, zoneID)
	if err == nil {
		return rulesetID, nil
	}

	// Create new rate limiting ruleset
	ruleset := cf.Ruleset{
		Name:        "Security Rules - Rate Limiting",
		Description: "Rate limiting security rules managed by kubeflare",
		Kind:        "zone",
		Phase:       "http_request_rate_limit",
		Rules:       []cf.RulesetRule{},
	}

	resp, err := c.api.CreateZoneRuleset(ctx, zoneID, ruleset)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// findRateLimitingRuleset finds the rate limiting ruleset for a zone
func (c *Client) findRateLimitingRuleset(ctx context.Context, zoneID string) (string, error) {
	rulesets, err := c.api.ListZoneRulesets(ctx, zoneID)
	if err != nil {
		return "", err
	}

	for _, ruleset := range rulesets {
		if ruleset.Phase == "http_request_rate_limit" {
			return ruleset.ID, nil
		}
	}

	return "", errors.New("rate limiting ruleset not found")
}

// convertToRulesetRule converts from CRD model to Ruleset API model
func convertToRulesetRule(rateLimit *v1alpha1.RateLimit) cf.RulesetRule {
	// Build expression from match criteria
	expression := buildRateLimitExpression(rateLimit.Spec.Match)

	// Prepare action parameters based on the action mode
	actionParams := &cf.RulesetRuleActionParameters{}
	if rateLimit.Spec.Threshold > 0 && rateLimit.Spec.Period > 0 {
		actionParams.Headers = map[string]cf.RulesetRuleActionParametersHTTPHeader{
			"X-Rate-Limit-Requests": {Value: fmt.Sprintf("%d", rateLimit.Spec.Threshold)},
			"X-Rate-Limit-Period":   {Value: fmt.Sprintf("%d", rateLimit.Spec.Period)},
		}
	}

	// Note: RateLimit and Response fields don't exist in RulesetRuleActionParameters
	// These need to be handled differently or the struct needs to be updated
	// For now, we'll use headers to pass rate limiting information

	return cf.RulesetRule{
		Expression:       expression,
		Action:           "rate_limit",
		Description:      rateLimit.Spec.Description,
		Enabled:          !rateLimit.Spec.Disabled,
		ActionParameters: actionParams,
	}
}

// convertFromRulesetRule converts from Ruleset API model to our internal model
func convertFromRulesetRule(rule *cf.RulesetRule) *RateLimitRule {
	rl := &RateLimitRule{
		ID:          rule.ID,
		Expression:  rule.Expression,
		Action:      rule.Action,
		Description: rule.Description,
		Enabled:     rule.Enabled,
	}

	// Extract rate limiting information from headers since RateLimit field doesn't exist
	if rule.ActionParameters != nil && rule.ActionParameters.Headers != nil {
		if reqHeader, ok := rule.ActionParameters.Headers["X-Rate-Limit-Requests"]; ok {
			if requests, err := fmt.Sscanf(reqHeader.Value, "%d", &rl.Requests); err == nil && requests == 1 {
				// Successfully parsed requests
			}
		}
		if periodHeader, ok := rule.ActionParameters.Headers["X-Rate-Limit-Period"]; ok {
			if periods, err := fmt.Sscanf(periodHeader.Value, "%d", &rl.Period); err == nil && periods == 1 {
				// Successfully parsed period
			}
		}
	}

	return rl
}

// buildRateLimitExpression builds a Cloudflare expression from match criteria
func buildRateLimitExpression(match v1alpha1.RateLimitMatch) string {
	var conditions []string

	// Add method conditions
	if len(match.Methods) > 0 {
		methodConditions := make([]string, len(match.Methods))
		for i, method := range match.Methods {
			methodConditions[i] = fmt.Sprintf(`http.request.method eq "%s"`, method)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(methodConditions, " or ")))
	}

	// Add scheme conditions
	if len(match.Schemes) > 0 {
		schemeConditions := make([]string, len(match.Schemes))
		for i, scheme := range match.Schemes {
			schemeConditions[i] = fmt.Sprintf(`http.request.uri.scheme eq "%s"`, scheme)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(schemeConditions, " or ")))
	}

	// Add URL pattern conditions
	if len(match.URL.Patterns) > 0 {
		urlConditions := make([]string, len(match.URL.Patterns))
		for i, pattern := range match.URL.Patterns {
			urlConditions[i] = fmt.Sprintf(`http.request.uri.path matches "%s"`, pattern)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(urlConditions, " or ")))
	}

	if len(conditions) == 0 {
		return "true" // Match all requests if no conditions specified
	}

	return strings.Join(conditions, " and ")
}

// ConvertFromCF converts from internal RateLimitRule to CRD model
func ConvertFromCF(rule *RateLimitRule) *v1alpha1.RateLimitSpec {
	spec := &v1alpha1.RateLimitSpec{
		Description: rule.Description,
		Disabled:    !rule.Enabled,
		Threshold:   rule.Requests,
		Period:      rule.Period,
	}

	// Note: Converting expression back to match criteria is complex
	// For now, we'll store the expression as-is and handle it in the CRD
	// You might want to add an Expression field to your CRD for full compatibility

	return spec
}
