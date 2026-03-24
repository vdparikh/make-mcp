package healing

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Engine handles self-healing for tool failures
type Engine struct {
	patterns []ErrorPattern
}

// ErrorPattern defines a known error pattern and its fix
type ErrorPattern struct {
	Pattern       *regexp.Regexp
	ErrorType     ErrorType
	SuggestionFn  func(err string, statusCode int, input json.RawMessage) *Suggestion
}

// ErrorType categorizes the type of error
type ErrorType string

const (
	ErrorTypeAuth       ErrorType = "authentication"
	ErrorTypeSchema     ErrorType = "schema_mismatch"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeServer     ErrorType = "server_error"
	ErrorTypeUnknown    ErrorType = "unknown"
)

// Suggestion represents a healing suggestion
type Suggestion struct {
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	AutoFix     bool                   `json:"auto_fix"`
	FixAction   string                 `json:"fix_action"`
	FixParams   map[string]interface{} `json:"fix_params,omitempty"`
	Confidence  float64                `json:"confidence"`
	Description string                 `json:"description"`
}

// Analysis contains the analysis of a tool failure
type Analysis struct {
	ErrorType      ErrorType     `json:"error_type"`
	RootCause      string        `json:"root_cause"`
	Suggestions    []*Suggestion `json:"suggestions"`
	CanAutoRepair  bool          `json:"can_auto_repair"`
	RepairStrategy string        `json:"repair_strategy,omitempty"`
}

// NewEngine creates a new healing engine
func NewEngine() *Engine {
	e := &Engine{}
	e.initPatterns()
	return e
}

func (e *Engine) initPatterns() {
	e.patterns = []ErrorPattern{
		{
			Pattern:   regexp.MustCompile(`(?i)(401|unauthorized|authentication.*failed|invalid.*token|token.*expired|jwt.*expired)`),
			ErrorType: ErrorTypeAuth,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				if strings.Contains(strings.ToLower(err), "expired") {
					return &Suggestion{
						Type:        "refresh_token",
						Message:     "Authentication token has expired",
						AutoFix:     true,
						FixAction:   "refresh_oauth_token",
						Confidence:  0.9,
						Description: "Automatically refresh the OAuth token and retry the request",
					}
				}
				return &Suggestion{
					Type:        "reauth",
					Message:     "Authentication failed - credentials may be invalid",
					AutoFix:     false,
					FixAction:   "prompt_reauth",
					Confidence:  0.8,
					Description: "User needs to re-authenticate with valid credentials",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(403|forbidden|permission.*denied|access.*denied|insufficient.*permissions)`),
			ErrorType: ErrorTypeAuth,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:        "permission",
					Message:     "Insufficient permissions for this operation",
					AutoFix:     false,
					FixAction:   "request_permission",
					Confidence:  0.85,
					Description: "Check if the user/application has required permissions",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(429|rate.*limit|too.*many.*requests|throttle|quota.*exceeded)`),
			ErrorType: ErrorTypeRateLimit,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:        "rate_limit",
					Message:     "Rate limit exceeded",
					AutoFix:     true,
					FixAction:   "retry_with_backoff",
					FixParams: map[string]interface{}{
						"initial_delay_ms": 1000,
						"max_retries":      3,
						"backoff_factor":   2,
					},
					Confidence:  0.95,
					Description: "Retry the request with exponential backoff",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(schema.*mismatch|invalid.*field|missing.*field|unexpected.*field|type.*error|json.*parse|unmarshal)`),
			ErrorType: ErrorTypeSchema,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				fieldMatch := regexp.MustCompile(`(?i)field\s*[:\s'"]*(\w+)`).FindStringSubmatch(err)
				fieldName := ""
				if len(fieldMatch) > 1 {
					fieldName = fieldMatch[1]
				}
				return &Suggestion{
					Type:    "schema_update",
					Message: fmt.Sprintf("Schema mismatch detected for field: %s", fieldName),
					AutoFix: false,
					FixAction: "update_schema",
					FixParams: map[string]interface{}{
						"detected_field": fieldName,
					},
					Confidence:  0.7,
					Description: "Update the tool schema to match the API response",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(timeout|deadline.*exceeded|context.*deadline|timed.*out)`),
			ErrorType: ErrorTypeTimeout,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:    "timeout",
					Message: "Request timed out",
					AutoFix: true,
					FixAction: "retry_with_extended_timeout",
					FixParams: map[string]interface{}{
						"timeout_ms":   30000,
						"max_retries":  2,
					},
					Confidence:  0.85,
					Description: "Retry with an extended timeout",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(404|not.*found|resource.*not.*found|endpoint.*not.*found)`),
			ErrorType: ErrorTypeNotFound,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:        "not_found",
					Message:     "Resource or endpoint not found",
					AutoFix:     false,
					FixAction:   "verify_endpoint",
					Confidence:  0.9,
					Description: "Verify the API endpoint URL is correct and the resource exists",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(500|502|503|504|internal.*server.*error|service.*unavailable|bad.*gateway)`),
			ErrorType: ErrorTypeServer,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:    "server_error",
					Message: "Server error - the external service may be experiencing issues",
					AutoFix: true,
					FixAction: "retry_with_backoff",
					FixParams: map[string]interface{}{
						"initial_delay_ms": 2000,
						"max_retries":      3,
						"backoff_factor":   2,
					},
					Confidence:  0.75,
					Description: "Retry the request as the server may recover",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(connection.*refused|network.*unreachable|dns.*resolution|no.*route.*to.*host)`),
			ErrorType: ErrorTypeNetwork,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:        "network",
					Message:     "Network connectivity issue",
					AutoFix:     true,
					FixAction:   "retry_with_backoff",
					FixParams: map[string]interface{}{
						"initial_delay_ms": 5000,
						"max_retries":      3,
						"backoff_factor":   2,
					},
					Confidence:  0.7,
					Description: "Retry after checking network connectivity",
				}
			},
		},
		{
			Pattern:   regexp.MustCompile(`(?i)(validation.*error|invalid.*input|bad.*request|malformed|required.*field)`),
			ErrorType: ErrorTypeValidation,
			SuggestionFn: func(err string, statusCode int, input json.RawMessage) *Suggestion {
				return &Suggestion{
					Type:        "validation",
					Message:     "Input validation failed",
					AutoFix:     false,
					FixAction:   "fix_input",
					Confidence:  0.8,
					Description: "Review and correct the input parameters",
				}
			},
		},
	}
}

// AnalyzeFailure analyzes a tool execution failure and suggests fixes
func (e *Engine) AnalyzeFailure(exec *models.ToolExecution) *Analysis {
	analysis := &Analysis{
		ErrorType:   ErrorTypeUnknown,
		Suggestions: make([]*Suggestion, 0),
	}

	if exec.Success {
		return analysis
	}

	errorStr := exec.Error
	if errorStr == "" && exec.StatusCode >= 400 {
		errorStr = fmt.Sprintf("HTTP %d error", exec.StatusCode)
	}

	for _, pattern := range e.patterns {
		if pattern.Pattern.MatchString(errorStr) || (exec.StatusCode > 0 && pattern.Pattern.MatchString(fmt.Sprintf("%d", exec.StatusCode))) {
			analysis.ErrorType = pattern.ErrorType
			suggestion := pattern.SuggestionFn(errorStr, exec.StatusCode, exec.Input)
			if suggestion != nil {
				analysis.Suggestions = append(analysis.Suggestions, suggestion)
				if suggestion.AutoFix {
					analysis.CanAutoRepair = true
					analysis.RepairStrategy = suggestion.FixAction
				}
			}
		}
	}

	if len(analysis.Suggestions) == 0 {
		analysis.RootCause = "Unknown error - manual investigation required"
		analysis.Suggestions = append(analysis.Suggestions, &Suggestion{
			Type:        "unknown",
			Message:     errorStr,
			AutoFix:     false,
			FixAction:   "manual_review",
			Confidence:  0.3,
			Description: "Error does not match known patterns - requires manual investigation",
		})
	} else {
		analysis.RootCause = analysis.Suggestions[0].Message
	}

	return analysis
}

// CreateHealingSuggestion creates a healing suggestion record from analysis
func (e *Engine) CreateHealingSuggestion(toolID string, analysis *Analysis) *models.HealingSuggestion {
	if len(analysis.Suggestions) == 0 {
		return nil
	}

	suggestion := analysis.Suggestions[0]
	suggestionJSON, _ := json.Marshal(suggestion)

	return &models.HealingSuggestion{
		ToolID:         toolID,
		ErrorPattern:   string(analysis.ErrorType),
		SuggestionType: suggestion.Type,
		Suggestion:     suggestionJSON,
		AutoApply:      suggestion.AutoFix && suggestion.Confidence >= 0.8,
		Applied:        false,
		CreatedAt:      time.Now(),
	}
}

// DetectPatternFromHistory analyzes execution history to detect recurring issues
func (e *Engine) DetectPatternFromHistory(executions []models.ToolExecution) []PatternDetection {
	if len(executions) < 3 {
		return nil
	}

	errorCounts := make(map[ErrorType]int)
	totalFailures := 0

	for _, exec := range executions {
		if !exec.Success {
			totalFailures++
			analysis := e.AnalyzeFailure(&exec)
			errorCounts[analysis.ErrorType]++
		}
	}

	var detections []PatternDetection
	for errType, count := range errorCounts {
		if count >= 3 || (totalFailures > 0 && float64(count)/float64(totalFailures) > 0.5) {
			detections = append(detections, PatternDetection{
				ErrorType:   errType,
				Occurrences: count,
				Percentage:  float64(count) / float64(len(executions)) * 100,
				Recommendation: e.getPatternRecommendation(errType, count),
			})
		}
	}

	return detections
}

// PatternDetection represents a detected error pattern
type PatternDetection struct {
	ErrorType      ErrorType `json:"error_type"`
	Occurrences    int       `json:"occurrences"`
	Percentage     float64   `json:"percentage"`
	Recommendation string    `json:"recommendation"`
}

func (e *Engine) getPatternRecommendation(errType ErrorType, count int) string {
	switch errType {
	case ErrorTypeAuth:
		return "Frequent authentication errors detected. Consider implementing automatic token refresh or reviewing credential management."
	case ErrorTypeRateLimit:
		return "Frequent rate limiting detected. Consider implementing request queuing or caching."
	case ErrorTypeTimeout:
		return "Frequent timeouts detected. Consider increasing timeout values or optimizing the API calls."
	case ErrorTypeSchema:
		return "Frequent schema mismatches detected. The external API may have changed - review and update the tool schema."
	case ErrorTypeServer:
		return "Frequent server errors detected. The external service may be unstable - consider adding a circuit breaker."
	case ErrorTypeNetwork:
		return "Frequent network errors detected. Consider implementing retry logic with exponential backoff."
	default:
		return fmt.Sprintf("Recurring errors detected (%d occurrences). Manual investigation recommended.", count)
	}
}
