package context

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Engine handles context injection for tool execution
type Engine struct {
	configs map[string][]models.ContextConfig
}

// NewEngine creates a new context engine
func NewEngine() *Engine {
	return &Engine{
		configs: make(map[string][]models.ContextConfig),
	}
}

// RegisterConfigs registers context configurations for a server
func (e *Engine) RegisterConfigs(serverID string, configs []models.ContextConfig) {
	e.configs[serverID] = configs
}

// ContextSource defines where context data comes from
type ContextSource string

const (
	SourceHeader   ContextSource = "header"
	SourceJWT      ContextSource = "jwt"
	SourceQuery    ContextSource = "query"
	SourceDatabase ContextSource = "database"
	SourceCustom   ContextSource = "custom"
)

// ExtractedContext contains context data extracted from a request
type ExtractedContext struct {
	UserID         string                 `json:"user_id,omitempty"`
	OrganizationID string                 `json:"organization_id,omitempty"`
	Permissions    []string               `json:"permissions,omitempty"`
	Roles          []string               `json:"roles,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	Custom         map[string]interface{} `json:"custom,omitempty"`
}

// HeaderConfig defines configuration for header-based context extraction
type HeaderConfig struct {
	HeaderName string `json:"header_name"`
	TargetField string `json:"target_field"`
}

// JWTConfig defines configuration for JWT-based context extraction
type JWTConfig struct {
	HeaderName string            `json:"header_name"`
	ClaimsMap  map[string]string `json:"claims_map"`
}

// InjectContext injects context into tool input based on configuration
func (e *Engine) InjectContext(serverID string, toolContextFields []string, input map[string]interface{}, req *http.Request) (map[string]interface{}, *ExtractedContext, error) {
	extracted := &ExtractedContext{
		Custom: make(map[string]interface{}),
	}

	configs, ok := e.configs[serverID]
	if !ok {
		return input, extracted, nil
	}

	for _, config := range configs {
		switch ContextSource(config.SourceType) {
		case SourceHeader:
			if err := e.extractFromHeader(config, req, extracted); err != nil {
				return nil, nil, fmt.Errorf("extracting header context: %w", err)
			}
		case SourceJWT:
			if err := e.extractFromJWT(config, req, extracted); err != nil {
				return nil, nil, fmt.Errorf("extracting JWT context: %w", err)
			}
		case SourceQuery:
			if err := e.extractFromQuery(config, req, extracted); err != nil {
				return nil, nil, fmt.Errorf("extracting query context: %w", err)
			}
		}
	}

	contextMap := e.buildContextMap(extracted, toolContextFields)
	if input == nil {
		input = make(map[string]interface{})
	}
	input["context"] = contextMap

	return input, extracted, nil
}

func (e *Engine) extractFromHeader(config models.ContextConfig, req *http.Request, ctx *ExtractedContext) error {
	if req == nil {
		return nil
	}

	var headerConfig HeaderConfig
	if err := json.Unmarshal(config.Config, &headerConfig); err != nil {
		return fmt.Errorf("parsing header config: %w", err)
	}

	value := req.Header.Get(headerConfig.HeaderName)
	if value == "" {
		return nil
	}

	switch headerConfig.TargetField {
	case "user_id":
		ctx.UserID = value
	case "organization_id":
		ctx.OrganizationID = value
	case "session_id":
		ctx.SessionID = value
	default:
		ctx.Custom[headerConfig.TargetField] = value
	}

	return nil
}

func (e *Engine) extractFromJWT(config models.ContextConfig, req *http.Request, ctx *ExtractedContext) error {
	if req == nil {
		return nil
	}

	var jwtConfig JWTConfig
	if err := json.Unmarshal(config.Config, &jwtConfig); err != nil {
		return fmt.Errorf("parsing JWT config: %w", err)
	}

	headerName := jwtConfig.HeaderName
	if headerName == "" {
		headerName = "Authorization"
	}

	authHeader := req.Header.Get(headerName)
	if authHeader == "" {
		return nil
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := e.parseJWTClaims(token)
	if err != nil {
		return nil
	}

	for claimKey, targetField := range jwtConfig.ClaimsMap {
		value, ok := claims[claimKey]
		if !ok {
			continue
		}

		switch targetField {
		case "user_id":
			if s, ok := value.(string); ok {
				ctx.UserID = s
			}
		case "organization_id":
			if s, ok := value.(string); ok {
				ctx.OrganizationID = s
			}
		case "permissions":
			if arr, ok := value.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						ctx.Permissions = append(ctx.Permissions, s)
					}
				}
			}
		case "roles":
			if arr, ok := value.([]interface{}); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						ctx.Roles = append(ctx.Roles, s)
					}
				}
			}
		default:
			ctx.Custom[targetField] = value
		}
	}

	return nil
}

func (e *Engine) parseJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload := parts[1]
	if l := len(payload) % 4; l > 0 {
		payload += strings.Repeat("=", 4-l)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("decoding JWT payload: %w", err)
		}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}

	return claims, nil
}

func (e *Engine) extractFromQuery(config models.ContextConfig, req *http.Request, ctx *ExtractedContext) error {
	if req == nil {
		return nil
	}

	var queryConfig HeaderConfig
	if err := json.Unmarshal(config.Config, &queryConfig); err != nil {
		return fmt.Errorf("parsing query config: %w", err)
	}

	value := req.URL.Query().Get(queryConfig.HeaderName)
	if value == "" {
		return nil
	}

	switch queryConfig.TargetField {
	case "user_id":
		ctx.UserID = value
	case "organization_id":
		ctx.OrganizationID = value
	default:
		ctx.Custom[queryConfig.TargetField] = value
	}

	return nil
}

func (e *Engine) buildContextMap(ctx *ExtractedContext, fields []string) map[string]interface{} {
	result := make(map[string]interface{})

	if len(fields) == 0 {
		if ctx.UserID != "" {
			result["user_id"] = ctx.UserID
		}
		if ctx.OrganizationID != "" {
			result["organization_id"] = ctx.OrganizationID
		}
		if len(ctx.Permissions) > 0 {
			result["permissions"] = ctx.Permissions
		}
		if len(ctx.Roles) > 0 {
			result["roles"] = ctx.Roles
		}
		for k, v := range ctx.Custom {
			result[k] = v
		}
		return result
	}

	for _, field := range fields {
		switch field {
		case "user_id":
			if ctx.UserID != "" {
				result["user_id"] = ctx.UserID
			}
		case "organization_id":
			if ctx.OrganizationID != "" {
				result["organization_id"] = ctx.OrganizationID
			}
		case "permissions":
			if len(ctx.Permissions) > 0 {
				result["permissions"] = ctx.Permissions
			}
		case "roles":
			if len(ctx.Roles) > 0 {
				result["roles"] = ctx.Roles
			}
		case "session_id":
			if ctx.SessionID != "" {
				result["session_id"] = ctx.SessionID
			}
		default:
			if v, ok := ctx.Custom[field]; ok {
				result[field] = v
			}
		}
	}

	return result
}

// ValidateContextForTool checks if required context fields are present
func (e *Engine) ValidateContextForTool(ctx *ExtractedContext, requiredFields []string) error {
	for _, field := range requiredFields {
		switch field {
		case "user_id":
			if ctx.UserID == "" {
				return fmt.Errorf("missing required context field: user_id")
			}
		case "organization_id":
			if ctx.OrganizationID == "" {
				return fmt.Errorf("missing required context field: organization_id")
			}
		case "permissions":
			if len(ctx.Permissions) == 0 {
				return fmt.Errorf("missing required context field: permissions")
			}
		case "roles":
			if len(ctx.Roles) == 0 {
				return fmt.Errorf("missing required context field: roles")
			}
		default:
			if _, ok := ctx.Custom[field]; !ok {
				return fmt.Errorf("missing required context field: %s", field)
			}
		}
	}
	return nil
}
