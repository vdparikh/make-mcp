package governance

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/vdparikh/make-mcp/backend/internal/context"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Engine handles AI governance and policy evaluation
type Engine struct {
	policies map[string][]models.Policy
}

// NewEngine creates a new governance engine
func NewEngine() *Engine {
	return &Engine{
		policies: make(map[string][]models.Policy),
	}
}

// RegisterPolicies registers policies for a tool
func (e *Engine) RegisterPolicies(toolID string, policies []models.Policy) {
	e.policies[toolID] = policies
}

// EvaluationResult contains the result of policy evaluation
type EvaluationResult struct {
	Allowed        bool                  `json:"allowed"`
	Reason         string                `json:"reason,omitempty"`
	ViolatedRules  []string              `json:"violated_rules,omitempty"`
	RequiresApproval bool                `json:"requires_approval"`
	ApprovalReason string                `json:"approval_reason,omitempty"`
}

// ApprovalRequiredConfig defines approval rule configuration
type ApprovalRequiredConfig struct {
	ApprovalType string `json:"approval_type"` // human, manager, system
	Message      string `json:"message"`
}

// MaxValueConfig defines max value rule configuration
type MaxValueConfig struct {
	Field    string  `json:"field"`
	MaxValue float64 `json:"max_value"`
}

// AllowedRolesConfig defines allowed roles rule configuration
type AllowedRolesConfig struct {
	Roles []string `json:"roles"`
}

// TimeWindowConfig defines time window rule configuration
type TimeWindowConfig struct {
	StartHour int      `json:"start_hour"` // 0-23
	EndHour   int      `json:"end_hour"`   // 0-23
	Timezone  string   `json:"timezone"`
	Weekdays  []int    `json:"weekdays"` // 0=Sunday, 6=Saturday
}

// RateLimitConfig defines rate limit rule configuration
type RateLimitConfig struct {
	MaxCalls   int    `json:"max_calls"`
	WindowSecs int    `json:"window_secs"`
	Scope      string `json:"scope"` // user, organization, global
}

// EvaluatePolicy evaluates all policies for a tool call
func (e *Engine) EvaluatePolicy(toolID string, input map[string]interface{}, ctx *context.ExtractedContext) *EvaluationResult {
	policies, ok := e.policies[toolID]
	if !ok || len(policies) == 0 {
		return &EvaluationResult{Allowed: true}
	}

	result := &EvaluationResult{Allowed: true}

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		for _, rule := range policy.Rules {
			ruleResult := e.evaluateRule(rule, input, ctx)
			if !ruleResult.Allowed {
				if rule.FailAction == "deny" {
					result.Allowed = false
					result.Reason = ruleResult.Reason
					result.ViolatedRules = append(result.ViolatedRules, fmt.Sprintf("%s: %s", rule.Type, ruleResult.Reason))
				} else if rule.FailAction == "approve" {
					result.RequiresApproval = true
					result.ApprovalReason = ruleResult.Reason
				}
			}
		}
	}

	return result
}

func (e *Engine) evaluateRule(rule models.PolicyRule, input map[string]interface{}, ctx *context.ExtractedContext) *EvaluationResult {
	switch rule.Type {
	case models.PolicyRuleApproval:
		return e.evaluateApprovalRule(rule, input)
	case models.PolicyRuleMaxValue:
		return e.evaluateMaxValueRule(rule, input)
	case models.PolicyRuleAllowedRoles:
		return e.evaluateAllowedRolesRule(rule, ctx)
	case models.PolicyRuleTimeWindow:
		return e.evaluateTimeWindowRule(rule)
	case models.PolicyRuleRateLimit:
		return e.evaluateRateLimitRule(rule, ctx)
	default:
		return &EvaluationResult{Allowed: true}
	}
}

func (e *Engine) evaluateApprovalRule(rule models.PolicyRule, input map[string]interface{}) *EvaluationResult {
	var config ApprovalRequiredConfig
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return &EvaluationResult{Allowed: true}
	}

	return &EvaluationResult{
		Allowed:        false,
		RequiresApproval: true,
		ApprovalReason: config.Message,
		Reason:         fmt.Sprintf("Approval required: %s", config.Message),
	}
}

func (e *Engine) evaluateMaxValueRule(rule models.PolicyRule, input map[string]interface{}) *EvaluationResult {
	var config MaxValueConfig
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return &EvaluationResult{Allowed: true}
	}

	value, ok := input[config.Field]
	if !ok {
		return &EvaluationResult{Allowed: true}
	}

	var numValue float64
	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case string:
		var err error
		numValue, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return &EvaluationResult{Allowed: true}
		}
	default:
		return &EvaluationResult{Allowed: true}
	}

	if numValue > config.MaxValue {
		return &EvaluationResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Value %v for field '%s' exceeds maximum allowed value of %v", numValue, config.Field, config.MaxValue),
		}
	}

	return &EvaluationResult{Allowed: true}
}

func (e *Engine) evaluateAllowedRolesRule(rule models.PolicyRule, ctx *context.ExtractedContext) *EvaluationResult {
	var config AllowedRolesConfig
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return &EvaluationResult{Allowed: true}
	}

	if ctx == nil || len(ctx.Roles) == 0 {
		return &EvaluationResult{
			Allowed: false,
			Reason:  "No roles found in context",
		}
	}

	for _, userRole := range ctx.Roles {
		for _, allowedRole := range config.Roles {
			if userRole == allowedRole {
				return &EvaluationResult{Allowed: true}
			}
		}
	}

	return &EvaluationResult{
		Allowed: false,
		Reason:  fmt.Sprintf("User roles %v not in allowed roles %v", ctx.Roles, config.Roles),
	}
}

func (e *Engine) evaluateTimeWindowRule(rule models.PolicyRule) *EvaluationResult {
	var config TimeWindowConfig
	if err := json.Unmarshal(rule.Config, &config); err != nil {
		return &EvaluationResult{Allowed: true}
	}

	loc := time.UTC
	if config.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(config.Timezone)
		if err != nil {
			loc = time.UTC
		}
	}

	now := time.Now().In(loc)
	hour := now.Hour()
	weekday := int(now.Weekday())

	if len(config.Weekdays) > 0 {
		weekdayAllowed := false
		for _, d := range config.Weekdays {
			if d == weekday {
				weekdayAllowed = true
				break
			}
		}
		if !weekdayAllowed {
			return &EvaluationResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Tool call not allowed on %s", now.Weekday().String()),
			}
		}
	}

	if config.StartHour <= config.EndHour {
		if hour < config.StartHour || hour >= config.EndHour {
			return &EvaluationResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Tool call not allowed at hour %d (allowed: %d-%d)", hour, config.StartHour, config.EndHour),
			}
		}
	} else {
		if hour < config.StartHour && hour >= config.EndHour {
			return &EvaluationResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Tool call not allowed at hour %d (allowed: %d-%d)", hour, config.StartHour, config.EndHour),
			}
		}
	}

	return &EvaluationResult{Allowed: true}
}

func (e *Engine) evaluateRateLimitRule(rule models.PolicyRule, ctx *context.ExtractedContext) *EvaluationResult {
	return &EvaluationResult{Allowed: true}
}

// CreatePolicyFromYAML creates a policy from YAML configuration
func (e *Engine) CreatePolicyFromYAML(yamlStr string) (*models.Policy, error) {
	return nil, fmt.Errorf("YAML parsing not implemented")
}

// ValidatePolicy validates a policy definition
func (e *Engine) ValidatePolicy(policy *models.Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.ToolID == "" {
		return fmt.Errorf("tool_id is required")
	}

	for i, rule := range policy.Rules {
		if rule.Type == "" {
			return fmt.Errorf("rule %d: type is required", i)
		}

		switch rule.Type {
		case models.PolicyRuleMaxValue:
			var config MaxValueConfig
			if err := json.Unmarshal(rule.Config, &config); err != nil {
				return fmt.Errorf("rule %d: invalid max_value config: %w", i, err)
			}
			if config.Field == "" {
				return fmt.Errorf("rule %d: max_value requires field", i)
			}
		case models.PolicyRuleAllowedRoles:
			var config AllowedRolesConfig
			if err := json.Unmarshal(rule.Config, &config); err != nil {
				return fmt.Errorf("rule %d: invalid allowed_roles config: %w", i, err)
			}
			if len(config.Roles) == 0 {
				return fmt.Errorf("rule %d: allowed_roles requires at least one role", i)
			}
		case models.PolicyRuleTimeWindow:
			var config TimeWindowConfig
			if err := json.Unmarshal(rule.Config, &config); err != nil {
				return fmt.Errorf("rule %d: invalid time_window config: %w", i, err)
			}
		case models.PolicyRuleRateLimit:
			var config RateLimitConfig
			if err := json.Unmarshal(rule.Config, &config); err != nil {
				return fmt.Errorf("rule %d: invalid rate_limit config: %w", i, err)
			}
			if config.MaxCalls <= 0 {
				return fmt.Errorf("rule %d: rate_limit requires positive max_calls", i)
			}
		}
	}

	return nil
}
