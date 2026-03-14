package security

import (
	"encoding/json"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Checklist reference: https://github.com/slowmist/MCP-Security-Checklist

const ChecklistURL = "https://github.com/slowmist/MCP-Security-Checklist"

// CriterionResult is one checklist item result.
type CriterionResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Priority string `json:"priority"` // "high" | "medium" | "low"
	Met      bool   `json:"met"`
	Reason   string `json:"reason,omitempty"`
}

// ScoreResult is the overall security score and per-criterion results.
type ScoreResult struct {
	Score      int              `json:"score"`      // 0-100
	Grade      string           `json:"grade"`      // A/B/C/D/F
	MaxPoints  int              `json:"max_points"`
	Earned     int              `json:"earned"`
	Criteria   []CriterionResult `json:"criteria"`
	ChecklistURL string         `json:"checklist_url"`
}

// weight by priority for scoring
const (
	weightHigh   = 3
	weightMedium = 2
	weightLow    = 1
)

// Score computes the security score for a server based on the SlowMist MCP Security Checklist.
// policiesByTool maps tool ID -> list of policies for that tool.
func Score(server *models.Server, policiesByTool map[string][]models.Policy) ScoreResult {
	var criteria []CriterionResult
	earned, maxPoints := 0, 0

	// Helper: add a criterion and update points
	add := func(id, name, priority string, met bool, reason string) {
		var w int
		switch priority {
		case "high":
			w = weightHigh
		case "medium":
			w = weightMedium
		default:
			w = weightLow
		}
		maxPoints += w
		if met {
			earned += w
		}
		criteria = append(criteria, CriterionResult{ID: id, Name: name, Priority: priority, Met: met, Reason: reason})
	}

	// --- Input validation (High): all tools have input_schema with type and properties/required
	allToolsHaveInputSchema := true
	for _, t := range server.Tools {
		if len(t.InputSchema) == 0 {
			allToolsHaveInputSchema = false
			break
		}
		var sch struct {
			Type       string                   `json:"type"`
			Properties map[string]interface{}    `json:"properties"`
			Required   []string                 `json:"required"`
		}
		_ = json.Unmarshal(t.InputSchema, &sch)
		if sch.Type != "object" || (len(sch.Properties) == 0 && len(sch.Required) == 0) {
			allToolsHaveInputSchema = false
			break
		}
	}
	if len(server.Tools) == 0 {
		allToolsHaveInputSchema = true
	}
	add("input_validation", "Input validation (tool schemas)", "high", allToolsHaveInputSchema,
		iff(!allToolsHaveInputSchema, "Define input_schema with type and properties for every tool"))

	// --- API rate limiting (Medium): at least one tool has a policy with rate_limit rule
	hasRateLimit := false
	for _, policies := range policiesByTool {
		for _, p := range policies {
			if !p.Enabled {
				continue
			}
			for _, r := range p.Rules {
				if r.Type == models.PolicyRuleRateLimit {
					hasRateLimit = true
					break
				}
			}
		}
	}
	if len(server.Tools) == 0 {
		hasRateLimit = true
	}
	add("rate_limiting", "API rate limiting", "medium", hasRateLimit,
		iff(!hasRateLimit, "Add a policy with Rate limit rule for tools that call external APIs"))

	// --- Access control (High): destructive tools have at least one policy
	destructiveHavePolicy := true
	for _, t := range server.Tools {
		if !t.DestructiveHint {
			continue
		}
		if len(policiesByTool[t.ID]) == 0 {
			destructiveHavePolicy = false
			break
		}
	}
	if len(server.Tools) == 0 {
		destructiveHavePolicy = true
	}
	add("access_control", "Access control (policies on destructive tools)", "high", destructiveHavePolicy,
		iff(!destructiveHavePolicy, "Attach at least one policy to every destructive tool"))

	// --- Least privilege / tool hints (Medium): at least one tool has read_only_hint or destructive_hint
	toolsAnnotated := false
	for _, t := range server.Tools {
		if t.DestructiveHint || t.ReadOnlyHint {
			toolsAnnotated = true
			break
		}
	}
	if len(server.Tools) == 0 {
		toolsAnnotated = true
	}
	add("least_privilege_hints", "Tool security hints (read-only / destructive)", "medium", toolsAnnotated,
		iff(!toolsAnnotated, "Mark tools as read-only or destructive in Tool Editor → Schema"))

	// --- Data isolation (High): server has owner (we always set it now)
	add("data_isolation", "Data isolation (owner-scoped)", "high", server.OwnerID != "",
		iff(server.OwnerID == "", "Server should be owner-scoped"))

	// --- Resource access control (High): resources have URI and handler
	resourcesControlled := true
	for _, r := range server.Resources {
		if r.URI == "" || len(r.Handler) == 0 {
			resourcesControlled = false
			break
		}
	}
	if len(server.Resources) == 0 {
		resourcesControlled = true
	}
	add("resource_access_control", "Resource access control", "high", resourcesControlled,
		iff(!resourcesControlled, "Define URI and handler for every resource"))

	// --- CLI allowlist (High): every CLI tool has allowed_commands
	cliHasAllowlist := true
	for _, t := range server.Tools {
		if t.ExecutionType != models.ExecutionTypeCLI {
			continue
		}
		if len(t.ExecutionConfig) == 0 {
			cliHasAllowlist = false
			break
		}
		var cfg struct {
			AllowedCommands []string `json:"allowed_commands"`
		}
		_ = json.Unmarshal(t.ExecutionConfig, &cfg)
		if len(cfg.AllowedCommands) == 0 {
			cliHasAllowlist = false
			break
		}
	}
	add("cli_allowlist", "CLI tool command allowlist", "high", cliHasAllowlist,
		iff(!cliHasAllowlist, "Set allowed_commands in execution config for every CLI tool"))

	// --- Container security (High): generated Dockerfile uses non-root (we always do)
	add("container_security", "Container security (non-root)", "high", true,
		"Generated Dockerfile runs as non-root user")

	// --- Tool permission control (High): destructive tools have policies (same as access control)
	add("tool_permission_control", "Tool permission control", "high", destructiveHavePolicy,
		iff(!destructiveHavePolicy, "Attach policies to destructive tools"))

	// --- Version / pinning (Medium): server has version
	hasVersion := server.Version != "" || server.LatestVersion != ""
	add("version_pinning", "Version / pinning", "medium", hasVersion,
		iff(!hasVersion, "Set server version and publish versions for pinning"))

	// --- Logging support (High): generated server supports file logging
	add("logging", "Logging support", "high", true,
		"Generated server supports MCP_LOG_FILE and run-with-log.mjs")

	// --- Role-based policy (Medium): at least one policy uses allowed_roles
	hasRolePolicy := false
	for _, policies := range policiesByTool {
		for _, p := range policies {
			if !p.Enabled {
				continue
			}
			for _, r := range p.Rules {
				if r.Type == models.PolicyRuleAllowedRoles {
					hasRolePolicy = true
					break
				}
			}
		}
	}
	add("rbac", "Role-based access (allowed_roles)", "medium", hasRolePolicy || len(server.Tools) == 0,
		iff(!hasRolePolicy && len(server.Tools) > 0, "Add a policy with Allowed roles for sensitive tools"))

	// Compute percentage and grade
	score := 0
	if maxPoints > 0 {
		score = (earned * 100) / maxPoints
	}
	grade := gradeFromScore(score)

	return ScoreResult{
		Score:        score,
		Grade:       grade,
		MaxPoints:   maxPoints,
		Earned:      earned,
		Criteria:    criteria,
		ChecklistURL: ChecklistURL,
	}
}

func iff(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}

func gradeFromScore(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}
