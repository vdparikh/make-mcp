package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vdparikh/make-mcp/backend/internal/context"
	"github.com/vdparikh/make-mcp/backend/internal/database"
	"github.com/vdparikh/make-mcp/backend/internal/generator"
	"github.com/vdparikh/make-mcp/backend/internal/governance"
	"github.com/vdparikh/make-mcp/backend/internal/healing"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Handler contains all API handlers
type Handler struct {
	db         *database.DB
	generator  *generator.Generator
	context    *context.Engine
	governance *governance.Engine
	healing    *healing.Engine
}

// NewHandler creates a new API handler
func NewHandler(db *database.DB) *Handler {
	return &Handler{
		db:         db,
		generator:  generator.NewGenerator(),
		context:    context.NewEngine(),
		governance: governance.NewEngine(),
		healing:    healing.NewEngine(),
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		servers := api.Group("/servers")
		{
			servers.GET("", h.ListServers)
			servers.POST("", h.CreateServer)
			servers.GET("/:id", h.GetServer)
			servers.PUT("/:id", h.UpdateServer)
			servers.DELETE("/:id", h.DeleteServer)
			servers.POST("/:id/generate", h.GenerateServer)
			servers.GET("/:id/context-configs", h.GetContextConfigs)
			servers.POST("/:id/context-configs", h.CreateContextConfig)
		}

		tools := api.Group("/tools")
		{
			tools.POST("", h.CreateTool)
			tools.GET("/:id", h.GetTool)
			tools.PUT("/:id", h.UpdateTool)
			tools.DELETE("/:id", h.DeleteTool)
			tools.POST("/:id/test", h.TestTool)
			tools.GET("/:id/executions", h.GetToolExecutions)
			tools.GET("/:id/policies", h.GetToolPolicies)
			tools.GET("/:id/healing", h.GetHealingSuggestions)
		}

		resources := api.Group("/resources")
		{
			resources.POST("", h.CreateResource)
			resources.DELETE("/:id", h.DeleteResource)
		}

		prompts := api.Group("/prompts")
		{
			prompts.POST("", h.CreatePrompt)
			prompts.DELETE("/:id", h.DeletePrompt)
		}

		policies := api.Group("/policies")
		{
			policies.POST("", h.CreatePolicy)
			policies.DELETE("/:id", h.DeletePolicy)
			policies.POST("/evaluate", h.EvaluatePolicy)
		}

		compositions := api.Group("/compositions")
		{
			compositions.GET("", h.ListCompositions)
			compositions.POST("", h.CreateComposition)
		}

		api.GET("/health", h.HealthCheck)
	}
}

// HealthCheck returns the health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// Server handlers
func (h *Handler) ListServers(c *gin.Context) {
	servers, err := h.db.ListServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, servers)
}

func (h *Handler) CreateServer(c *gin.Context) {
	var req models.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.db.CreateServer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

func (h *Handler) GetServer(c *gin.Context) {
	id := c.Param("id")

	server, err := h.db.GetServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if server == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *Handler) UpdateServer(c *gin.Context) {
	id := c.Param("id")

	var req models.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.db.UpdateServer(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

func (h *Handler) DeleteServer(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GenerateServer(c *gin.Context) {
	id := c.Param("id")

	server, err := h.db.GetServer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if server == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	zipData, err := h.generator.GenerateZip(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-mcp-server.zip", server.Name))
	c.Data(http.StatusOK, "application/zip", zipData)
}

// Tool handlers
func (h *Handler) CreateTool(c *gin.Context) {
	var req models.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tool, err := h.db.CreateTool(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tool)
}

func (h *Handler) GetTool(c *gin.Context) {
	id := c.Param("id")

	tool, err := h.db.GetTool(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tool == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
		return
	}

	c.JSON(http.StatusOK, tool)
}

func (h *Handler) UpdateTool(c *gin.Context) {
	id := c.Param("id")

	var req models.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tool, err := h.db.UpdateTool(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tool)
}

func (h *Handler) DeleteTool(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteTool(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) TestTool(c *gin.Context) {
	id := c.Param("id")

	tool, err := h.db.GetTool(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tool == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
		return
	}

	var req models.TestToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var inputMap map[string]interface{}
	if req.Input != nil {
		if err := json.Unmarshal(req.Input, &inputMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input JSON"})
			return
		}
	}

	configs, _ := h.db.GetContextConfigs(c.Request.Context(), tool.ServerID)
	h.context.RegisterConfigs(tool.ServerID, configs)

	inputMap, extractedCtx, err := h.context.InjectContext(tool.ServerID, tool.ContextFields, inputMap, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policies, _ := h.db.GetPoliciesByTool(c.Request.Context(), tool.ID)
	h.governance.RegisterPolicies(tool.ID, policies)

	policyResult := h.governance.EvaluatePolicy(tool.ID, inputMap, extractedCtx)
	if !policyResult.Allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"error":          "Policy violation",
			"reason":         policyResult.Reason,
			"violated_rules": policyResult.ViolatedRules,
		})
		return
	}

	if policyResult.RequiresApproval {
		c.JSON(http.StatusAccepted, gin.H{
			"status":          "approval_required",
			"approval_reason": policyResult.ApprovalReason,
		})
		return
	}

	start := time.Now()
	result, statusCode, execErr := h.executeTool(tool, inputMap)
	duration := time.Since(start).Milliseconds()

	exec := &models.ToolExecution{
		ToolID:     tool.ID,
		ServerID:   tool.ServerID,
		Input:      req.Input,
		DurationMs: duration,
		StatusCode: statusCode,
		Success:    execErr == nil && statusCode >= 200 && statusCode < 300,
	}

	if execErr != nil {
		exec.Error = execErr.Error()
	}

	if result != nil {
		exec.Output, _ = json.Marshal(result)
	}

	h.db.LogToolExecution(c.Request.Context(), exec)

	if !exec.Success {
		analysis := h.healing.AnalyzeFailure(exec)

		if analysis.CanAutoRepair {
			suggestion := h.healing.CreateHealingSuggestion(tool.ID, analysis)
			if suggestion != nil {
				h.db.CreateHealingSuggestion(c.Request.Context(), suggestion)
			}
		}

		c.JSON(http.StatusOK, models.TestToolResponse{
			Success:  false,
			Error:    exec.Error,
			Duration: duration,
			Output:   json.RawMessage(fmt.Sprintf(`{"healing_analysis": %s}`, mustMarshal(analysis))),
		})
		return
	}

	c.JSON(http.StatusOK, models.TestToolResponse{
		Success:  true,
		Output:   exec.Output,
		Duration: duration,
	})
}

func (h *Handler) executeTool(tool *models.Tool, input map[string]interface{}) (interface{}, int, error) {
	switch tool.ExecutionType {
	case models.ExecutionTypeRestAPI:
		return h.executeRestAPI(tool, input)
	case models.ExecutionTypeWebhook:
		return h.executeWebhook(tool, input)
	default:
		return map[string]interface{}{
			"message": "Tool executed successfully (mock)",
			"input":   input,
		}, 200, nil
	}
}

func (h *Handler) executeRestAPI(tool *models.Tool, input map[string]interface{}) (interface{}, int, error) {
	var config struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	}

	if err := json.Unmarshal(tool.ExecutionConfig, &config); err != nil {
		return nil, 0, fmt.Errorf("invalid execution config: %w", err)
	}

	if config.Method == "" {
		config.Method = "GET"
	}

	url := config.URL
	for key, value := range input {
		url = replaceTemplate(url, key, fmt.Sprintf("%v", value))
	}

	var body io.Reader
	if config.Method != "GET" {
		jsonBody, _ := json.Marshal(input)
		body = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(config.Method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		result = string(respBody)
	}

	if resp.StatusCode >= 400 {
		return result, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return result, resp.StatusCode, nil
}

func (h *Handler) executeWebhook(tool *models.Tool, input map[string]interface{}) (interface{}, int, error) {
	var config struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}

	if err := json.Unmarshal(tool.ExecutionConfig, &config); err != nil {
		return nil, 0, fmt.Errorf("invalid execution config: %w", err)
	}

	jsonBody, _ := json.Marshal(input)
	req, err := http.NewRequest("POST", config.URL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		result = string(respBody)
	}

	return result, resp.StatusCode, nil
}

func (h *Handler) GetToolExecutions(c *gin.Context) {
	id := c.Param("id")

	executions, err := h.db.GetToolExecutions(c.Request.Context(), id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range executions {
		if !executions[i].Success {
			analysis := h.healing.AnalyzeFailure(&executions[i])
			if analysis != nil && len(analysis.Suggestions) > 0 {
				executions[i].Error = fmt.Sprintf("%s [Analysis: %s]", executions[i].Error, analysis.RootCause)
			}
		}
	}

	c.JSON(http.StatusOK, executions)
}

func (h *Handler) GetHealingSuggestions(c *gin.Context) {
	id := c.Param("id")

	suggestions, err := h.db.GetHealingSuggestions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

// Resource handlers
func (h *Handler) CreateResource(c *gin.Context) {
	var req models.CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resource, err := h.db.CreateResource(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resource)
}

func (h *Handler) DeleteResource(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteResource(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Prompt handlers
func (h *Handler) CreatePrompt(c *gin.Context) {
	var req models.CreatePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prompt, err := h.db.CreatePrompt(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, prompt)
}

func (h *Handler) DeletePrompt(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeletePrompt(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// Policy handlers
func (h *Handler) CreatePolicy(c *gin.Context) {
	var req models.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy, err := h.db.CreatePolicy(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

func (h *Handler) GetToolPolicies(c *gin.Context) {
	id := c.Param("id")

	policies, err := h.db.GetPoliciesByTool(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, policies)
}

func (h *Handler) DeletePolicy(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeletePolicy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) EvaluatePolicy(c *gin.Context) {
	var req struct {
		ToolID  string                 `json:"tool_id" binding:"required"`
		Input   map[string]interface{} `json:"input"`
		Context map[string]interface{} `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policies, _ := h.db.GetPoliciesByTool(c.Request.Context(), req.ToolID)
	h.governance.RegisterPolicies(req.ToolID, policies)

	ctx := &context.ExtractedContext{
		Custom: make(map[string]interface{}),
	}
	if req.Context != nil {
		if v, ok := req.Context["user_id"].(string); ok {
			ctx.UserID = v
		}
		if v, ok := req.Context["organization_id"].(string); ok {
			ctx.OrganizationID = v
		}
		if v, ok := req.Context["roles"].([]interface{}); ok {
			for _, r := range v {
				if s, ok := r.(string); ok {
					ctx.Roles = append(ctx.Roles, s)
				}
			}
		}
	}

	result := h.governance.EvaluatePolicy(req.ToolID, req.Input, ctx)
	c.JSON(http.StatusOK, result)
}

// Context Config handlers
func (h *Handler) GetContextConfigs(c *gin.Context) {
	id := c.Param("id")

	configs, err := h.db.GetContextConfigs(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

func (h *Handler) CreateContextConfig(c *gin.Context) {
	serverID := c.Param("id")

	var req struct {
		Name       string          `json:"name" binding:"required"`
		SourceType string          `json:"source_type" binding:"required"`
		Config     json.RawMessage `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := h.db.CreateContextConfig(c.Request.Context(), serverID, req.Name, req.SourceType, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// Composition handlers
func (h *Handler) ListCompositions(c *gin.Context) {
	compositions, err := h.db.ListCompositions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, compositions)
}

func (h *Handler) CreateComposition(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		ServerIDs   []string `json:"server_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	composition, err := h.db.CreateComposition(c.Request.Context(), req.Name, req.Description, req.ServerIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, composition)
}

func replaceTemplate(s, key string, value string) string {
	patterns := []string{
		fmt.Sprintf("{{%s}}", key),
		fmt.Sprintf("{%s}", key),
		fmt.Sprintf(":%s", key),
	}

	result := s
	for _, pattern := range patterns {
		result = strings.ReplaceAll(result, pattern, value)
	}
	return result
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

