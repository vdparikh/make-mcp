package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// IngestObservabilityEvents receives tool execution events from a running MCP server (key in body).
func (h *Handler) IngestObservabilityEvents(c *gin.Context) {
	var req models.ObservabilityEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}
	if len(req.Events) == 0 {
		c.JSON(http.StatusOK, gin.H{"accepted": 0})
		return
	}
	server, err := h.db.GetServerByObservabilityKey(c.Request.Context(), req.Key)
	if err != nil || server == nil {
		fmt.Printf("IngestObservabilityEvents: rejected request (unknown key), events=%d\n", len(req.Events))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or unknown observability key"})
		return
	}
	ctx := c.Request.Context()
	toolNames := make([]string, 0, len(req.Events))
	for _, ev := range req.Events {
		if n := strings.TrimSpace(ev.ToolName); n != "" {
			toolNames = append(toolNames, n)
		}
	}
	toolByName, toolErr := h.db.GetToolIDsByServerAndNames(ctx, server.ID, toolNames)
	if toolErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": toolErr.Error()})
		return
	}
	accepted := 0
	for _, ev := range req.Events {
		toolName := strings.TrimSpace(ev.ToolName)
		if toolName == "" {
			continue
		}
		toolID := toolByName[toolName]
		if toolID == "" {
			// Snapshot / hosted / marketplace rows can diverge from tools table; still record the event by name.
			fmt.Printf("IngestObservabilityEvents: server=%s tool_name=%q has no matching tools row; logging with null tool_id\n", server.ID, toolName)
		}
		exec := &models.ToolExecution{
			ToolID:           toolID,
			ServerID:         server.ID,
			ToolName:         ev.ToolName,
			Source:           "runtime",
			ClientUserID:     strings.TrimSpace(ev.ClientUserID),
			ClientAgent:      strings.TrimSpace(ev.ClientAgent),
			ClientToken:      strings.TrimSpace(ev.ClientToken),
			DurationMs:       ev.DurationMs,
			Success:          ev.Success,
			Error:            ev.Error,
			RepairSuggestion: ev.RepairSuggestion,
		}
		if !ev.Success && ev.RepairSuggestion == "" {
			analysis := h.healing.AnalyzeFailure(exec)
			if analysis != nil && len(analysis.Suggestions) > 0 {
				exec.RepairSuggestion = analysis.Suggestions[0].Message
			}
		}
		if err := h.db.LogToolExecution(ctx, exec); err != nil {
			continue
		}
		accepted++
	}
	fmt.Printf("IngestObservabilityEvents: server=%s accepted=%d total=%d\n", server.ID, accepted, len(req.Events))
	c.JSON(http.StatusOK, gin.H{"accepted": accepted})
}

// GetServerObservability returns recent runtime events, latency and failure stats, and repair suggestions.
func (h *Handler) GetServerObservability(c *gin.Context) {
	serverID := c.Param("id")
	server := h.requireServerCoreOwnership(c, serverID)
	if server == nil {
		return
	}
	events, err := h.db.ListRuntimeExecutionsByServer(c.Request.Context(), serverID, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latencyList, failureList, repairSuggestions := aggregateObservabilityExecutionStats(events)
	scheme := "https"
	if c.GetHeader("X-Forwarded-Proto") == "http" || c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + c.Request.Host
	resp := models.ObservabilitySummaryResponse{
		ReportingKey:      server.ObservabilityReportingKey,
		EndpointURL:       baseURL + "/api/observability/events",
		RecentEvents:      events,
		LatencyByTool:     latencyList,
		FailuresByTool:    failureList,
		RepairSuggestions: repairSuggestions,
	}
	c.JSON(http.StatusOK, resp)
}

// GetObservability returns runtime events and stats for the current user, with optional server_id and tool_name filters.
func (h *Handler) GetObservability(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	serverID := strings.TrimSpace(c.Query("server_id"))
	toolName := strings.TrimSpace(c.Query("tool_name"))
	clientUserID := strings.TrimSpace(c.Query("client_user_id"))
	clientAgent := strings.TrimSpace(c.Query("client_agent"))
	limit := 200
	if l := c.Query("limit"); l != "" {
		if n, err := fmt.Sscanf(l, "%d", &limit); n == 1 && err == nil && limit > 0 && limit <= 500 {
			// use parsed limit
		} else {
			limit = 200
		}
	}
	ctx := c.Request.Context()
	serverSummaries, err := h.db.ListServerSummariesByOwner(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	events, err := h.db.ListRuntimeExecutionsForUser(ctx, userID, serverID, toolName, clientUserID, clientAgent, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latencyList, failureList, repairSuggestions := aggregateObservabilityExecutionStats(events)
	c.JSON(http.StatusOK, models.ObservabilityDashboardResponse{
		Servers:           serverSummaries,
		RecentEvents:      events,
		LatencyByTool:     latencyList,
		FailuresByTool:    failureList,
		RepairSuggestions: repairSuggestions,
	})
}

// EnableServerObservability generates or returns the reporting key and shows env vars for the generated server.
func (h *Handler) EnableServerObservability(c *gin.Context) {
	serverID := c.Param("id")
	if h.requireServerCoreOwnership(c, serverID) == nil {
		return
	}
	key, err := h.db.EnsureServerObservabilityKey(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scheme := "https"
	if c.GetHeader("X-Forwarded-Proto") == "http" || c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + c.Request.Host
	c.JSON(http.StatusOK, gin.H{
		"key":          key,
		"endpoint_url": baseURL + "/api/observability/events",
		"env": gin.H{
			"MCP_OBSERVABILITY_ENDPOINT": baseURL + "/api/observability/events",
			"MCP_OBSERVABILITY_KEY":      key,
		},
	})
}
