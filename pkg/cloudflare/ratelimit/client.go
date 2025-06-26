package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/replicatedhq/kubeflare/pkg/apis/crds/v1alpha1"
)

// ClientInterface defines the interface for rate limit operations
type ClientInterface interface {
	Create(ctx context.Context, rateLimit *v1alpha1.RateLimit) (string, error)
	Get(ctx context.Context, zoneID, ruleID string) (*RateLimitRule, error)
	Update(ctx context.Context, rateLimit *v1alpha1.RateLimit) error
	Delete(ctx context.Context, zoneID, ruleID string) error
	List(ctx context.Context, zoneID string) ([]*RateLimitRule, error)
}

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

	// Update the entire ruleset with the new rule using new API format
	rc := cf.ZoneIdentifier(rateLimit.Spec.ZoneID)
	ruleset, err := c.api.GetRuleset(ctx, rc, rulesetID)
	if err != nil {
		return "", fmt.Errorf("failed to get ruleset: %w", err)
	}

	// Add the new rule to the ruleset
	newRule := rule
	newRule.ID = "" // Let Cloudflare generate the ID
	ruleset.Rules = append(ruleset.Rules, newRule)

	updateParams := cf.UpdateRulesetParams{
		ID:          rulesetID,
		Description: ruleset.Description,
		Rules:       ruleset.Rules,
	}
	resp, err := c.api.UpdateRuleset(ctx, rc, updateParams)
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

	rc := cf.ZoneIdentifier(zoneID)
	ruleset, err := c.api.GetRuleset(ctx, rc, rulesetID)
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
	rc := cf.ZoneIdentifier(rateLimit.Spec.ZoneID)
	ruleset, err := c.api.GetRuleset(ctx, rc, rulesetID)
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
	updateParams := cf.UpdateRulesetParams{
		ID:          rulesetID,
		Description: ruleset.Description,
		Rules:       ruleset.Rules,
	}
	_, err = c.api.UpdateRuleset(ctx, rc, updateParams)
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
	rc := cf.ZoneIdentifier(zoneID)
	ruleset, err := c.api.GetRuleset(ctx, rc, rulesetID)
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
	updateParams := cf.UpdateRulesetParams{
		ID:          rulesetID,
		Description: ruleset.Description,
		Rules:       filteredRules,
	}
	_, err = c.api.UpdateRuleset(ctx, rc, updateParams)
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

	rc := cf.ZoneIdentifier(zoneID)
	ruleset, err := c.api.GetRuleset(ctx, rc, rulesetID)
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
		Phase:       "http_ratelimit",
		Rules:       []cf.RulesetRule{},
	}

	rc := cf.ZoneIdentifier(zoneID)
	createParams := cf.CreateRulesetParams{
		Name:        ruleset.Name,
		Description: ruleset.Description,
		Kind:        ruleset.Kind,
		Phase:       ruleset.Phase,
		Rules:       ruleset.Rules,
	}
	resp, err := c.api.CreateRuleset(ctx, rc, createParams)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// findRateLimitingRuleset finds the rate limiting ruleset for a zone
func (c *Client) findRateLimitingRuleset(ctx context.Context, zoneID string) (string, error) {
	rc := cf.ZoneIdentifier(zoneID)
	listParams := cf.ListRulesetsParams{}
	rulesets, err := c.api.ListRulesets(ctx, rc, listParams)
	if err != nil {
		return "", err
	}

	for _, ruleset := range rulesets {
		if ruleset.Phase == "http_ratelimit" {
			return ruleset.ID, nil
		}
	}

	return "", errors.New("rate limiting ruleset not found")
}

// mapActionMode converts kubeflare action modes to official Cloudflare API actions
func mapActionMode(mode string) string {
	switch mode {
	case "simulate":
		return "log" // Legacy: simulate maps to log
	case "ban":
		return "block" // Legacy: ban maps to block
	case "block", "challenge", "js_challenge", "managed_challenge", "log":
		return mode // Official Cloudflare actions
	default:
		return "log" // Default to log for unknown actions
	}
}

// convertToRulesetRule converts from CRD model to Ruleset API model
func convertToRulesetRule(rateLimit *v1alpha1.RateLimit) cf.RulesetRule {
	// Build expression from match criteria
	expression := buildRateLimitExpression(rateLimit.Spec.Match)

	// Map the action mode to official Cloudflare API action
	action := mapActionMode(rateLimit.Spec.Action.Mode)

	// Create enabled pointer
	enabled := !rateLimit.Spec.Disabled

	// Create the rule with proper rate limiting configuration using latest SDK
	rule := cf.RulesetRule{
		Expression:  expression,
		Action:      action,
		Description: rateLimit.Spec.Description,
		Enabled:     &enabled,
		RateLimit: &cf.RulesetRuleRateLimit{
			Characteristics:   []string{"cf.colo.id", "ip.src"}, // Include required cf.colo.id
			Period:            rateLimit.Spec.Period,
			RequestsPerPeriod: rateLimit.Spec.Threshold,
			MitigationTimeout: rateLimit.Spec.Action.Timeout,
		},
	}

	return rule
}

// convertFromRulesetRule converts from Ruleset API model to our internal model
func convertFromRulesetRule(rule *cf.RulesetRule) *RateLimitRule {
	enabled := rule.Enabled != nil && *rule.Enabled

	rl := &RateLimitRule{
		ID:          rule.ID,
		Expression:  rule.Expression,
		Action:      rule.Action,
		Description: rule.Description,
		Enabled:     enabled,
	}

	// Extract rate limiting information from RateLimit field in latest SDK
	if rule.RateLimit != nil {
		rl.Requests = rule.RateLimit.RequestsPerPeriod
		rl.Period = rule.RateLimit.Period
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
			schemeConditions[i] = fmt.Sprintf(`http.request.scheme eq "%s"`, scheme)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(schemeConditions, " or ")))
	}

	// Add URL pattern conditions using wildcard format like working example
	if len(match.URL.Patterns) > 0 {
		urlConditions := make([]string, len(match.URL.Patterns))
		for i, pattern := range match.URL.Patterns {
			// Convert pattern to path-only format and use wildcard operator
			// Remove domain prefix if present and ensure it starts with /
			pathPattern := pattern
			if strings.Contains(pattern, "/") {
				parts := strings.SplitN(pattern, "/", 2)
				if len(parts) > 1 {
					pathPattern = "/" + parts[1]
				}
			}
			urlConditions[i] = fmt.Sprintf(`starts_with(http.request.uri.path, "%s")`, strings.TrimSuffix(pathPattern, "*"))
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
