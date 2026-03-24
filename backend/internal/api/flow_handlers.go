package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vdparikh/make-mcp/backend/internal/mcpvalidate"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// ExecuteFlow runs a flow and returns the result
func (h *Handler) ExecuteFlow(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Input   json.RawMessage `json:"input"`
		Context json.RawMessage `json:"context"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow, err := h.db.GetFlow(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	nodes, edges, err := parseFlowGraph(flow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Execute the flow (simplified execution)
	result, nodeResults := h.executeFlowNodes(nodes, edges, req.Input, req.Context)

	c.JSON(http.StatusOK, models.FlowExecutionResponse{
		Success:     result.Success,
		Output:      result.Output,
		Error:       result.Error,
		Duration:    result.Duration,
		NodeResults: nodeResults,
	})
}

func (h *Handler) executeFlowNodes(nodes []models.FlowNode, edges []models.FlowEdge, input, ctx json.RawMessage) (models.FlowExecutionResponse, []models.NodeResult) {
	_ = ctx // reserved for future context-aware execution
	startTime := time.Now()
	nodeResults := []models.NodeResult{}

	// Build adjacency list
	outgoing := make(map[string][]string)
	for _, edge := range edges {
		outgoing[edge.Source] = append(outgoing[edge.Source], edge.Target)
	}

	// Find trigger node (start)
	var triggerNode *models.FlowNode
	for i := range nodes {
		if nodes[i].Type == "trigger" {
			triggerNode = &nodes[i]
			break
		}
	}

	if triggerNode == nil {
		return models.FlowExecutionResponse{
			Success:  false,
			Error:    "No trigger node found",
			Duration: time.Since(startTime).Milliseconds(),
		}, nodeResults
	}

	// Node map for quick lookup
	nodeMap := make(map[string]*models.FlowNode)
	for i := range nodes {
		nodeMap[nodes[i].ID] = &nodes[i]
	}

	// BFS execution
	currentData := input
	queue := []string{triggerNode.ID}
	visited := make(map[string]bool)
	var finalOutput json.RawMessage
	var lastError string

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		if visited[nodeID] {
			continue
		}
		visited[nodeID] = true

		node := nodeMap[nodeID]
		if node == nil {
			continue
		}

		nodeStart := time.Now()
		var nodeOutput json.RawMessage
		var nodeErr string
		success := true

		// Execute based on node type
		switch node.Type {
		case "trigger":
			nodeOutput = currentData
		case "api":
			// Parse node config and execute API call
			var config struct {
				URL     string            `json:"url"`
				Method  string            `json:"method"`
				Headers map[string]string `json:"headers"`
			}
			json.Unmarshal(node.Data, &config)

			// Simple simulation - in real implementation would make HTTP call
			nodeOutput, _ = json.Marshal(map[string]interface{}{
				"status": "simulated",
				"config": config,
				"input":  json.RawMessage(currentData),
			})
		case "cli":
			// Parse CLI config
			var config struct {
				Command string `json:"command"`
			}
			json.Unmarshal(node.Data, &config)

			nodeOutput, _ = json.Marshal(map[string]interface{}{
				"status":  "simulated",
				"command": config.Command,
			})
		case "transform":
			// Pass through with transformation note
			nodeOutput = currentData
		case "output":
			finalOutput = currentData
			nodeOutput = currentData
		default:
			nodeOutput = currentData
		}

		nodeResults = append(nodeResults, models.NodeResult{
			NodeID:   nodeID,
			NodeType: node.Type,
			Success:  success,
			Output:   nodeOutput,
			Error:    nodeErr,
			Duration: time.Since(nodeStart).Milliseconds(),
		})

		if !success {
			lastError = nodeErr
			break
		}

		// Update current data for next node
		currentData = nodeOutput

		// Add connected nodes to queue
		for _, nextID := range outgoing[nodeID] {
			queue = append(queue, nextID)
		}
	}

	return models.FlowExecutionResponse{
		Success:  lastError == "",
		Output:   finalOutput,
		Error:    lastError,
		Duration: time.Since(startTime).Milliseconds(),
	}, nodeResults
}

// ConvertFlowToTool converts a flow to a tool configuration
func (h *Handler) ConvertFlowToTool(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ToolName    string `json:"tool_name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow, err := h.db.GetFlow(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	nodes, edges, err := parseFlowGraph(flow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build input schema from trigger node
	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	// Extract input schema from trigger node if defined.
	if triggerSchema := extractTriggerInputSchema(nodes); triggerSchema != nil {
		inputSchema = triggerSchema
	}

	// Store the ENTIRE flow structure in execution config
	// This preserves all nodes, edges, and their configurations
	executionConfig := map[string]interface{}{
		"flow_id":          flow.ID,
		"flow_name":        flow.Name,
		"flow_description": flow.Description,
		"nodes":            nodes,
		"edges":            edges,
	}

	inputSchemaJSON, _ := json.Marshal(inputSchema)
	outputSchemaJSON, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"success":      map[string]interface{}{"type": "boolean"},
			"result":       map[string]interface{}{"type": "object"},
			"node_results": map[string]interface{}{"type": "array"},
		},
	})
	execConfigJSON, _ := json.Marshal(executionConfig)

	// Create the tool with "flow" execution type
	toolReq := models.CreateToolRequest{
		ServerID:        flow.ServerID,
		Name:            req.ToolName,
		Description:     req.Description,
		ExecutionType:   models.ExecutionTypeFlow,
		InputSchema:     inputSchemaJSON,
		OutputSchema:    outputSchemaJSON,
		ExecutionConfig: execConfigJSON,
	}

	tool, err := h.db.CreateTool(c.Request.Context(), toolReq)
	if err != nil {
		if errors.Is(err, mcpvalidate.ErrInvalidToolName) || errors.Is(err, mcpvalidate.ErrDuplicateToolName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tool":    tool,
		"message": "Flow converted to tool successfully",
	})
}

func (h *Handler) ListFlows(c *gin.Context) {
	flows, err := h.db.ListFlows(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flows)
}

func (h *Handler) CreateFlow(c *gin.Context) {
	var req models.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow, err := h.db.CreateFlow(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, flow)
}

func (h *Handler) GetFlow(c *gin.Context) {
	id := c.Param("id")
	flow, err := h.db.GetFlow(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flow)
}

func (h *Handler) UpdateFlow(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	flow, err := h.db.UpdateFlow(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flow)
}

func (h *Handler) DeleteFlow(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteFlow(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) GetServerFlows(c *gin.Context) {
	serverID := c.Param("id")
	if h.requireServerOwnership(c, serverID) == nil {
		return
	}
	flows, err := h.db.GetFlowsByServer(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, flows)
}

func parseFlowGraph(flow *models.Flow) ([]models.FlowNode, []models.FlowEdge, error) {
	var nodes []models.FlowNode
	var edges []models.FlowEdge
	if err := json.Unmarshal(flow.Nodes, &nodes); err != nil {
		return nil, nil, fmt.Errorf("invalid nodes format")
	}
	if err := json.Unmarshal(flow.Edges, &edges); err != nil {
		return nil, nil, fmt.Errorf("invalid edges format")
	}
	return nodes, edges, nil
}

func extractTriggerInputSchema(nodes []models.FlowNode) map[string]interface{} {
	for _, node := range nodes {
		if node.Type != "trigger" {
			continue
		}
		var data map[string]interface{}
		if err := json.Unmarshal(node.Data, &data); err != nil {
			return nil
		}
		raw, ok := data["inputSchema"]
		if !ok {
			return nil
		}
		schema, ok := raw.(map[string]interface{})
		if !ok {
			return nil
		}
		return schema
	}
	return nil
}
