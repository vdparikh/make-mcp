package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vdparikh/make-mcp/backend/internal/auth"
	"github.com/vdparikh/make-mcp/backend/internal/context"
	"github.com/vdparikh/make-mcp/backend/internal/database"
	"github.com/vdparikh/make-mcp/backend/internal/generator"
	"github.com/vdparikh/make-mcp/backend/internal/governance"
	"github.com/vdparikh/make-mcp/backend/internal/healing"
	"github.com/vdparikh/make-mcp/backend/internal/models"
	"github.com/vdparikh/make-mcp/backend/internal/openapi"
	"github.com/vdparikh/make-mcp/backend/internal/security"
	webauthnlib "github.com/go-webauthn/webauthn/webauthn"
	webauthnpkg "github.com/vdparikh/make-mcp/backend/internal/webauthn"
)

// Handler contains all API handlers
type Handler struct {
	db            *database.DB
	generator     *generator.Generator
	context       *context.Engine
	governance    *governance.Engine
	healing       *healing.Engine
	openapiParser *openapi.Parser
	webauthn      *webauthnlib.WebAuthn
	sessionStore  *webauthnpkg.SessionStore
}

// NewHandler creates a new API handler
func NewHandler(db *database.DB, wa *webauthnlib.WebAuthn, sessionStore *webauthnpkg.SessionStore) *Handler {
	return &Handler{
		db:            db,
		generator:     generator.NewGenerator(),
		context:       context.NewEngine(),
		governance:    governance.NewEngine(),
		healing:       healing.NewEngine(),
		openapiParser: openapi.NewParser(),
		webauthn:      wa,
		sessionStore:  sessionStore,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Auth routes (no authentication required)
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", h.Register)
			authGroup.POST("/webauthn/register/begin", h.WebAuthnRegisterBegin)
			authGroup.POST("/webauthn/register/finish", h.WebAuthnRegisterFinish)
			authGroup.POST("/webauthn/login/begin", h.WebAuthnLoginBegin)
			authGroup.POST("/webauthn/login/finish", h.WebAuthnLoginFinish)
			authGroup.GET("/me", h.AuthMiddleware(), h.GetCurrentUser)
		}

		servers := api.Group("/servers")
		servers.Use(h.AuthMiddleware())
		{
			servers.GET("", h.ListServers)
			servers.POST("", h.CreateServer)
			servers.POST("/demo", h.CreateDemoServer)
			servers.GET("/:id", h.GetServer)
			servers.PUT("/:id", h.UpdateServer)
			servers.DELETE("/:id", h.DeleteServer)
			servers.POST("/:id/generate", h.GenerateServer)
			servers.POST("/:id/github-export", h.GitHubExport)
			servers.GET("/:id/context-configs", h.GetContextConfigs)
			servers.POST("/:id/context-configs", h.CreateContextConfig)
			servers.POST("/:id/publish", h.PublishServer)
			servers.GET("/:id/versions", h.GetServerVersions)
			servers.GET("/:id/versions/:version", h.GetServerVersionSnapshot)
			servers.GET("/:id/versions/:version/download", h.DownloadServerVersion)
			servers.GET("/:id/flows", h.GetServerFlows)
			servers.GET("/:id/security-score", h.GetSecurityScore)
			servers.GET("/:id/observability", h.GetServerObservability)
			servers.POST("/:id/observability/enable", h.EnableServerObservability)
		}

		// Observability: dashboard (auth) and ingestion (key-based, no JWT)
		api.GET("/observability", h.AuthMiddleware(), h.GetObservability)
		api.POST("/observability/events", h.IngestObservabilityEvents)

		marketplace := api.Group("/marketplace")
		{
			marketplace.GET("", h.ListMarketplace)
			marketplace.GET("/:id", h.GetMarketplaceServer)
			marketplace.GET("/:id/download", h.DownloadMarketplaceServer)
		}

		contextConfigs := api.Group("/context-configs")
		{
			contextConfigs.DELETE("/:id", h.DeleteContextConfig)
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
		compositions.Use(h.AuthMiddleware())
		{
			compositions.GET("", h.ListCompositions)
			compositions.POST("", h.CreateComposition)
			compositions.GET("/:id", h.GetComposition)
			compositions.PUT("/:id", h.UpdateComposition)
			compositions.DELETE("/:id", h.DeleteComposition)
			compositions.POST("/:id/export", h.ExportComposition)
		}

		// Flows
		flows := api.Group("/flows")
		{
			flows.GET("", h.ListFlows)
			flows.POST("", h.CreateFlow)
			flows.GET("/:id", h.GetFlow)
			flows.PUT("/:id", h.UpdateFlow)
			flows.DELETE("/:id", h.DeleteFlow)
			flows.POST("/:id/execute", h.ExecuteFlow)
			flows.POST("/:id/convert", h.ConvertFlowToTool)
		}

		// OpenAPI import (auth required; created server is owned by current user)
		api.POST("/import/openapi", h.AuthMiddleware(), h.ImportOpenAPI)
		api.POST("/import/openapi/preview", h.PreviewOpenAPIImport)

		api.GET("/health", h.HealthCheck)
	}
}

// HealthCheck returns the health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// currentUserID returns the authenticated user ID from context (set by AuthMiddleware). Empty if not set.
func (h *Handler) currentUserID(c *gin.Context) string {
	v, _ := c.Get("userID")
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// requireServerOwnership loads the server by id and returns it only if the current user owns it; otherwise writes 403/404 and returns nil.
func (h *Handler) requireServerOwnership(c *gin.Context, serverID string) *models.Server {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil
	}
	server, err := h.db.GetServer(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil
	}
	if server == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return nil
	}
	if server.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this server"})
		return nil
	}
	return server
}

// userIDFromRequest returns the authenticated user ID from context or by parsing the Bearer token.
// Use this when the handler must have the correct user (e.g. ListServers).
func (h *Handler) userIDFromRequest(c *gin.Context) string {
	if id := strings.TrimSpace(h.currentUserID(c)); id != "" {
		return id
	}
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	claims, err := auth.ValidateToken(parts[1])
	if err != nil {
		return ""
	}
	id := strings.TrimSpace(claims.UserID)
	if id == "" && claims.Subject != "" {
		id = strings.TrimSpace(claims.Subject)
	}
	return id
}

// Server handlers (all require auth; list/create scoped to user; get/update/delete require ownership)
func (h *Handler) ListServers(c *gin.Context) {
	userID := h.userIDFromRequest(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	servers, err := h.db.ListServers(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("X-List-Count", fmt.Sprintf("%d", len(servers)))
	c.JSON(http.StatusOK, servers)
}

func (h *Handler) CreateServer(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req models.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.OwnerID = userID

	server, err := h.db.CreateServer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

func (h *Handler) CreateDemoServer(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	server, err := h.db.CreateDemoServerForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, server)
}

func (h *Handler) GetServer(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	c.JSON(http.StatusOK, server)
}

func (h *Handler) UpdateServer(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
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
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	if err := h.db.DeleteServer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GenerateServer(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	zipData, err := h.generator.GenerateZip(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-mcp-server.zip", generator.ServerSlug(server.Name)))
	c.Data(http.StatusOK, "application/zip", zipData)
}

// GitHubExport exports the server to a GitHub repository
func (h *Handler) GitHubExport(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Token       string `json:"token" binding:"required"`
		Owner       string `json:"owner" binding:"required"`
		Repo        string `json:"repo" binding:"required"`
		Branch      string `json:"branch"`
		CommitMsg   string `json:"commit_message"`
		CreateRepo  bool   `json:"create_repo"`
		Private     bool   `json:"private"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.CommitMsg == "" {
		req.CommitMsg = "Initial MCP server export from MCP Builder"
	}

	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	// Generate server files
	gen, err := h.generator.Generate(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("generating server: %s", err.Error())})
		return
	}

	// GitHub API client
	client := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://api.github.com"

	// Helper function for GitHub API requests
	githubRequest := func(method, endpoint string, body interface{}) (*http.Response, error) {
		var reqBody io.Reader
		if body != nil {
			jsonBody, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			reqBody = bytes.NewReader(jsonBody)
		}

		httpReq, err := http.NewRequest(method, baseURL+endpoint, reqBody)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+req.Token)
		httpReq.Header.Set("Accept", "application/vnd.github+json")
		httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if body != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
		return client.Do(httpReq)
	}

	// Check if repo exists
	repoEndpoint := fmt.Sprintf("/repos/%s/%s", req.Owner, req.Repo)
	resp, err := githubRequest("GET", repoEndpoint, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("checking repo: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	repoExists := resp.StatusCode == 200

	// Track the actual branch to use
	actualBranch := req.Branch

	// Create repo if needed
	if !repoExists && req.CreateRepo {
		createRepoBody := map[string]interface{}{
			"name":        req.Repo,
			"description": req.Description,
			"private":     req.Private,
			"auto_init":   true, // Create with README to have a branch
		}

		resp, err := githubRequest("POST", "/user/repos", createRepoBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating repo: %s", err.Error())})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			body, _ := io.ReadAll(resp.Body)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create repo: %s", string(body))})
			return
		}

		// Wait for GitHub to initialize the repo
		time.Sleep(3 * time.Second)

		// Get the actual default branch from the repo
		repoInfoResp, err := githubRequest("GET", repoEndpoint, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("getting repo info: %s", err.Error())})
			return
		}
		defer repoInfoResp.Body.Close()

		var repoInfo struct {
			DefaultBranch string `json:"default_branch"`
		}
		if err := json.NewDecoder(repoInfoResp.Body).Decode(&repoInfo); err == nil && repoInfo.DefaultBranch != "" {
			actualBranch = repoInfo.DefaultBranch
		}
	} else if !repoExists && !req.CreateRepo {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found. Enable 'Create repository' to create it."})
		return
	} else if repoExists {
		// For existing repos, get the default branch if user didn't specify one
		resp.Body.Close()
		repoInfoResp, err := githubRequest("GET", repoEndpoint, nil)
		if err == nil {
			var repoInfo struct {
				DefaultBranch string `json:"default_branch"`
			}
			if json.NewDecoder(repoInfoResp.Body).Decode(&repoInfo) == nil && repoInfo.DefaultBranch != "" {
				// Only use default if user didn't explicitly set a branch
				if req.Branch == "main" || req.Branch == "" {
					actualBranch = repoInfo.DefaultBranch
				}
			}
			repoInfoResp.Body.Close()
		}
	}

	// Get the branch SHA with retry for newly created repos
	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}

	refEndpoint := fmt.Sprintf("/repos/%s/%s/git/ref/heads/%s", req.Owner, req.Repo, actualBranch)
	
	// Retry a few times for newly created repos
	var refResp *http.Response
	for i := 0; i < 5; i++ {
		refResp, err = githubRequest("GET", refEndpoint, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("getting branch ref: %s", err.Error())})
			return
		}

		if refResp.StatusCode == 200 {
			break
		}
		refResp.Body.Close()
		
		if i < 4 {
			time.Sleep(2 * time.Second)
		}
	}

	if refResp.StatusCode != 200 {
		body, _ := io.ReadAll(refResp.Body)
		refResp.Body.Close()
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("branch '%s' not found: %s", actualBranch, string(body))})
		return
	}

	if err := json.NewDecoder(refResp.Body).Decode(&refData); err != nil {
		refResp.Body.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parsing branch ref"})
		return
	}
	refResp.Body.Close()

	// Get the commit tree
	commitEndpoint := fmt.Sprintf("/repos/%s/%s/git/commits/%s", req.Owner, req.Repo, refData.Object.SHA)
	resp, err = githubRequest("GET", commitEndpoint, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("getting commit: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	var commitData struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commitData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parsing commit data"})
		return
	}

	// Create blobs for each file
	type treeEntry struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
	}

	var treeEntries []treeEntry

	for path, content := range gen.Files {
		blobBody := map[string]string{
			"content":  base64.StdEncoding.EncodeToString(content),
			"encoding": "base64",
		}

		blobEndpoint := fmt.Sprintf("/repos/%s/%s/git/blobs", req.Owner, req.Repo)
		resp, err := githubRequest("POST", blobEndpoint, blobBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating blob for %s: %s", path, err.Error())})
			return
		}

		var blobData struct {
			SHA string `json:"sha"`
		}

		if resp.StatusCode != 201 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create blob: %s", string(body))})
			return
		}

		if err := json.NewDecoder(resp.Body).Decode(&blobData); err != nil {
			resp.Body.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "parsing blob response"})
			return
		}
		resp.Body.Close()

		treeEntries = append(treeEntries, treeEntry{
			Path: path,
			Mode: "100644",
			Type: "blob",
			SHA:  blobData.SHA,
		})
	}

	// Create new tree
	treeBody := map[string]interface{}{
		"base_tree": commitData.Tree.SHA,
		"tree":      treeEntries,
	}

	treeEndpoint := fmt.Sprintf("/repos/%s/%s/git/trees", req.Owner, req.Repo)
	resp, err = githubRequest("POST", treeEndpoint, treeBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating tree: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	var newTreeData struct {
		SHA string `json:"sha"`
	}

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create tree: %s", string(body))})
		return
	}

	if err := json.NewDecoder(resp.Body).Decode(&newTreeData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parsing tree response"})
		return
	}

	// Create new commit
	newCommitBody := map[string]interface{}{
		"message": req.CommitMsg,
		"tree":    newTreeData.SHA,
		"parents": []string{refData.Object.SHA},
	}

	newCommitEndpoint := fmt.Sprintf("/repos/%s/%s/git/commits", req.Owner, req.Repo)
	resp, err = githubRequest("POST", newCommitEndpoint, newCommitBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("creating commit: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	var newCommitData struct {
		SHA     string `json:"sha"`
		HTMLURL string `json:"html_url"`
	}

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create commit: %s", string(body))})
		return
	}

	if err := json.NewDecoder(resp.Body).Decode(&newCommitData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parsing commit response"})
		return
	}

	// Update branch reference
	updateRefBody := map[string]interface{}{
		"sha":   newCommitData.SHA,
		"force": false,
	}

	updateRefEndpoint := fmt.Sprintf("/repos/%s/%s/git/refs/heads/%s", req.Owner, req.Repo, actualBranch)
	resp, err = githubRequest("PATCH", updateRefEndpoint, updateRefBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("updating ref: %s", err.Error())})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to update ref: %s", string(body))})
		return
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s", req.Owner, req.Repo)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"repo_url":   repoURL,
		"commit_sha": newCommitData.SHA,
		"files":      len(gen.Files),
		"message":    fmt.Sprintf("Successfully pushed %d files to %s", len(gen.Files), repoURL),
	})
}

// PublishServer publishes a server version to the marketplace
func (h *Handler) PublishServer(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	var req models.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Check if version already exists
	existing, err := h.db.GetServerVersion(c.Request.Context(), id, req.Version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("version %s already exists", req.Version)})
		return
	}

	// Create snapshot of current server state
	snapshot, err := json.Marshal(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create snapshot"})
		return
	}

	// Get user ID from context (if authenticated)
	publishedBy := ""
	if userID, exists := c.Get("userID"); exists {
		publishedBy = userID.(string)
	}

	// Publish version
	version, err := h.db.PublishServerVersion(c.Request.Context(), id, req.Version, req.ReleaseNotes, publishedBy, snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update public status if requested
	if req.IsPublic {
		if err := h.db.UpdateServerPublicStatus(c.Request.Context(), id, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, version)
}

// GetServerVersions returns all published versions for a server
func (h *Handler) GetServerVersions(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	versions, err := h.db.GetServerVersions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

// GetServerVersionSnapshot returns the snapshot for a specific version
func (h *Handler) GetServerVersionSnapshot(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	version := c.Param("version")
	v, err := h.db.GetServerVersion(c.Request.Context(), id, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	// Parse snapshot and return full server
	var server models.Server
	if err := json.Unmarshal(v.Snapshot, &server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse snapshot"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"version":       v,
		"server":        server,
	})
}

// DownloadServerVersion generates and downloads a specific server version
func (h *Handler) DownloadServerVersion(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	version := c.Param("version")
	v, err := h.db.GetServerVersion(c.Request.Context(), id, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	// Parse snapshot
	var server models.Server
	if err := json.Unmarshal(v.Snapshot, &server); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse snapshot"})
		return
	}

	// Generate zip from snapshot
	zipData, err := h.generator.GenerateZip(&server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-v%s-mcp-server.zip", generator.ServerSlug(server.Name), version))
	c.Data(http.StatusOK, "application/zip", zipData)
}

// ListMarketplace returns all publicly published servers with security score (from snapshot; no policy data).
func (h *Handler) ListMarketplace(c *gin.Context) {
	servers, err := h.db.ListPublishedServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for i := range servers {
		result := security.Score(&servers[i], nil)
		servers[i].SecurityScore = &result.Score
		servers[i].SecurityGrade = &result.Grade
	}
	c.JSON(http.StatusOK, servers)
}

// GetMarketplaceServer returns a published server with its versions
func (h *Handler) GetMarketplaceServer(c *gin.Context) {
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

	if server.Status != models.ServerStatusPublished || !server.IsPublic {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	versions, err := h.db.GetServerVersions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Published security score (no policy data for other users' servers)
	scoreResult := security.Score(server, nil)
	server.SecurityScore = &scoreResult.Score
	server.SecurityGrade = &scoreResult.Grade

	c.JSON(http.StatusOK, gin.H{
		"server":         server,
		"versions":       versions,
		"security_score": scoreResult,
	})
}

// DownloadMarketplaceServer downloads the latest version of a marketplace server
func (h *Handler) DownloadMarketplaceServer(c *gin.Context) {
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

	if server.Status != models.ServerStatusPublished || !server.IsPublic {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not available"})
		return
	}

	// Get latest version
	if server.LatestVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no published version available"})
		return
	}

	v, err := h.db.GetServerVersion(c.Request.Context(), id, server.LatestVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	// Parse snapshot
	var snapshotServer models.Server
	if err := json.Unmarshal(v.Snapshot, &snapshotServer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse snapshot"})
		return
	}

	// Generate zip from snapshot
	zipData, err := h.generator.GenerateZip(&snapshotServer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Increment download counter
	_ = h.db.IncrementServerDownloads(c.Request.Context(), id)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-v%s-mcp-server.zip", generator.ServerSlug(snapshotServer.Name), server.LatestVersion))
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

// mergeSimulatedContext merges request-body context (e.g. from UI test) into extractedCtx so policy and tool see it.
func mergeSimulatedContext(sim map[string]interface{}, extractedCtx *context.ExtractedContext) {
	if extractedCtx.Custom == nil {
		extractedCtx.Custom = make(map[string]interface{})
	}
	for k, v := range sim {
		switch k {
		case "user_id":
			if s, ok := v.(string); ok {
				extractedCtx.UserID = s
			}
		case "organization_id":
			if s, ok := v.(string); ok {
				extractedCtx.OrganizationID = s
			}
		case "session_id":
			if s, ok := v.(string); ok {
				extractedCtx.SessionID = s
			}
		case "permissions":
			extractedCtx.Permissions = sliceOfStrings(v)
		case "roles":
			extractedCtx.Roles = sliceOfStrings(v)
		default:
			extractedCtx.Custom[k] = v
		}
	}
}

func sliceOfStrings(v interface{}) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
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

	// Allow request body context for simulation in the UI (overrides/supplements header/JWT-extracted context)
	if len(req.Context) > 0 {
		var sim map[string]interface{}
		if err := json.Unmarshal(req.Context, &sim); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid context JSON"})
			return
		}
		mergeSimulatedContext(sim, extractedCtx)
		if inputMap == nil {
			inputMap = make(map[string]interface{})
		}
		contextMap, _ := inputMap["context"].(map[string]interface{})
		if contextMap == nil {
			contextMap = make(map[string]interface{})
		}
		for k, v := range sim {
			contextMap[k] = v
		}
		inputMap["context"] = contextMap
	}

	policies, _ := h.db.GetPoliciesByTool(c.Request.Context(), tool.ID)
	h.governance.RegisterPolicies(tool.ID, policies)

	policyResult := h.governance.EvaluatePolicy(tool.ID, inputMap, extractedCtx)
	injectedForResponse, _ := inputMap["context"].(map[string]interface{})
	if !policyResult.Allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"error":            "Policy violation",
			"reason":           policyResult.Reason,
			"violated_rules":   policyResult.ViolatedRules,
			"injected_context": injectedForResponse,
		})
		return
	}

	if policyResult.RequiresApproval {
		c.JSON(http.StatusAccepted, gin.H{
			"status":           "approval_required",
			"approval_reason":  policyResult.ApprovalReason,
			"injected_context": injectedForResponse,
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
		out := result
		switch tool.OutputDisplay {
		case models.OutputDisplayTable:
			if wrapped := wrapToolOutputForMCPApp(result); wrapped != nil {
				out = wrapped
			}
		case models.OutputDisplayCard:
			if wrapped := wrapToolOutputForMCPAppCard(result); wrapped != nil {
				out = wrapped
			}
		}
		exec.Output, _ = json.Marshal(out)
	}

	h.db.LogToolExecution(c.Request.Context(), exec)

	injectedContext, _ := inputMap["context"].(map[string]interface{})

	if !exec.Success {
		analysis := h.healing.AnalyzeFailure(exec)

		if analysis.CanAutoRepair {
			suggestion := h.healing.CreateHealingSuggestion(tool.ID, analysis)
			if suggestion != nil {
				h.db.CreateHealingSuggestion(c.Request.Context(), suggestion)
			}
		}

		c.JSON(http.StatusOK, models.TestToolResponse{
			Success:         false,
			Error:           exec.Error,
			Duration:        duration,
			Output:          json.RawMessage(fmt.Sprintf(`{"healing_analysis": %s}`, mustMarshal(analysis))),
			InjectedContext: injectedContext,
		})
		return
	}

	c.JSON(http.StatusOK, models.TestToolResponse{
		Success:         true,
		Output:          exec.Output,
		Duration:        duration,
		InjectedContext: injectedContext,
	})
}

// wrapToolOutputForMCPApp wraps tool output in MCP Apps format for table widget.
// Handles: (1) array of objects → multi-row table, (2) single object → one-row table.
// Returns nil if result is not object(s). Conventional clients can use "text" fallback.
func wrapToolOutputForMCPApp(result interface{}) interface{} {
	var rows []map[string]interface{}
	var columns []map[string]interface{}
	seenKeys := make(map[string]bool)

	switch v := result.(type) {
	case []interface{}:
		if len(v) == 0 {
			return nil
		}
		for _, item := range v {
			obj, ok := item.(map[string]interface{})
			if !ok {
				return nil
			}
			rows = append(rows, obj)
			for k := range obj {
				seenKeys[k] = true
			}
		}
	case map[string]interface{}:
		rows = []map[string]interface{}{v}
		for k := range v {
			seenKeys[k] = true
		}
	default:
		return nil
	}

	keys := make([]string, 0, len(seenKeys))
	for k := range seenKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		columns = append(columns, map[string]interface{}{"key": k, "label": k})
	}

	textFallback, _ := json.Marshal(result)
	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "table",
			"props": map[string]interface{}{
				"columns": columns,
				"rows":    rows,
			},
		},
	}
}

// Preferred keys for card main content (first match wins)
var cardContentKeys = []string{"joke", "text", "content", "message", "body", "description", "quote"}

// wrapToolOutputForMCPAppCard wraps a single object as MCP App card (e.g. for joke/quote APIs).
// Extracts a main content string (joke, text, content, etc.) and optional title.
func wrapToolOutputForMCPAppCard(result interface{}) interface{} {
	obj, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}

	var content string
	for _, key := range cardContentKeys {
		if v, exists := obj[key]; exists {
			if s, ok := v.(string); ok && s != "" {
				content = s
				break
			}
		}
	}
	if content == "" {
		for k, v := range obj {
			if s, ok := v.(string); ok && len(s) > len(content) {
				content = s
				_ = k
			}
		}
	}
	if content == "" {
		return nil
	}

	title := "Result"
	if v, ok := obj["title"].(string); ok && v != "" {
		title = v
	} else if v, ok := obj["name"].(string); ok && v != "" {
		title = v
	}

	textFallback, _ := json.Marshal(result)
	return map[string]interface{}{
		"text": string(textFallback),
		"_mcp_app": map[string]interface{}{
			"widget": "card",
			"props": map[string]interface{}{
				"content": content,
				"title":   title,
			},
		},
	}
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or unknown observability key"})
		return
	}
	ctx := c.Request.Context()
	accepted := 0
	for _, ev := range req.Events {
		toolID, err := h.db.GetToolIDByServerAndName(ctx, server.ID, ev.ToolName)
		if err != nil || toolID == "" {
			continue
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
	c.JSON(http.StatusOK, gin.H{"accepted": accepted})
}

// GetServerObservability returns recent runtime events, latency and failure stats, and repair suggestions.
func (h *Handler) GetServerObservability(c *gin.Context) {
	serverID := c.Param("id")
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	server, err := h.db.GetServer(c.Request.Context(), serverID)
	if err != nil || server == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	if server.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	events, err := h.db.ListRuntimeExecutionsByServer(c.Request.Context(), serverID, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latencyByTool := make(map[string]*models.ToolLatencyStat)
	failuresByTool := make(map[string]*models.ToolFailureStat)
	var repairSuggestions []models.RepairSuggestionItem
	for i := range events {
		e := &events[i]
		toolKey := e.ToolName
		if toolKey == "" {
			toolKey = e.ToolID
		}
		if stat, ok := latencyByTool[toolKey]; ok {
			stat.Count++
			stat.AvgMs = (stat.AvgMs*float64(stat.Count-1) + float64(e.DurationMs)) / float64(stat.Count)
			if e.DurationMs > stat.P95Ms {
				stat.P95Ms = e.DurationMs
			}
		} else {
			latencyByTool[toolKey] = &models.ToolLatencyStat{ToolName: e.ToolName, ToolID: e.ToolID, Count: 1, AvgMs: float64(e.DurationMs), P95Ms: e.DurationMs}
		}
		if !e.Success {
			if f, ok := failuresByTool[toolKey]; ok {
				f.Count++
				if e.Error != "" {
					f.LastError = e.Error
				}
			} else {
				failuresByTool[toolKey] = &models.ToolFailureStat{ToolName: e.ToolName, ToolID: e.ToolID, Count: 1, LastError: e.Error}
			}
			if e.RepairSuggestion != "" {
				repairSuggestions = append(repairSuggestions, models.RepairSuggestionItem{
					ToolName: e.ToolName, ToolID: e.ToolID, Suggestion: e.RepairSuggestion, CreatedAt: e.CreatedAt,
				})
			}
		}
	}
	// P95: sort latencies per tool and take 95th percentile (simplified: use max for now; could sort and index)
	latencyList := make([]models.ToolLatencyStat, 0, len(latencyByTool))
	for _, s := range latencyByTool {
		latencyList = append(latencyList, *s)
	}
	failureList := make([]models.ToolFailureStat, 0, len(failuresByTool))
	for _, f := range failuresByTool {
		failureList = append(failureList, *f)
	}
	sort.Slice(repairSuggestions, func(i, j int) bool { return repairSuggestions[j].CreatedAt.Before(repairSuggestions[i].CreatedAt) })
	if len(repairSuggestions) > 20 {
		repairSuggestions = repairSuggestions[:20]
	}
	scheme := "https"
	if c.GetHeader("X-Forwarded-Proto") == "http" || c.Request.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + c.Request.Host
	resp := models.ObservabilitySummaryResponse{
		ReportingKey:       server.ObservabilityReportingKey,
		EndpointURL:        baseURL + "/api/observability/events",
		RecentEvents:       events,
		LatencyByTool:      latencyList,
		FailuresByTool:     failureList,
		RepairSuggestions:  repairSuggestions,
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
	servers, err := h.db.ListServers(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	serverSummaries := make([]models.ServerSummary, 0, len(servers))
	for _, s := range servers {
		serverSummaries = append(serverSummaries, models.ServerSummary{ID: s.ID, Name: s.Name})
	}
	events, err := h.db.ListRuntimeExecutionsForUser(ctx, userID, serverID, toolName, clientUserID, clientAgent, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	latencyByTool := make(map[string]*models.ToolLatencyStat)
	failuresByTool := make(map[string]*models.ToolFailureStat)
	var repairSuggestions []models.RepairSuggestionItem
	for i := range events {
		e := &events[i]
		toolKey := e.ToolName
		if toolKey == "" {
			toolKey = e.ToolID
		}
		if stat, ok := latencyByTool[toolKey]; ok {
			stat.Count++
			stat.AvgMs = (stat.AvgMs*float64(stat.Count-1) + float64(e.DurationMs)) / float64(stat.Count)
			if e.DurationMs > stat.P95Ms {
				stat.P95Ms = e.DurationMs
			}
		} else {
			latencyByTool[toolKey] = &models.ToolLatencyStat{ToolName: e.ToolName, ToolID: e.ToolID, Count: 1, AvgMs: float64(e.DurationMs), P95Ms: e.DurationMs}
		}
		if !e.Success {
			if f, ok := failuresByTool[toolKey]; ok {
				f.Count++
				if e.Error != "" {
					f.LastError = e.Error
				}
			} else {
				failuresByTool[toolKey] = &models.ToolFailureStat{ToolName: e.ToolName, ToolID: e.ToolID, Count: 1, LastError: e.Error}
			}
			if e.RepairSuggestion != "" {
				repairSuggestions = append(repairSuggestions, models.RepairSuggestionItem{
					ToolName: e.ToolName, ToolID: e.ToolID, Suggestion: e.RepairSuggestion, CreatedAt: e.CreatedAt,
				})
			}
		}
	}
	latencyList := make([]models.ToolLatencyStat, 0, len(latencyByTool))
	for _, s := range latencyByTool {
		latencyList = append(latencyList, *s)
	}
	failureList := make([]models.ToolFailureStat, 0, len(failuresByTool))
	for _, f := range failuresByTool {
		failureList = append(failureList, *f)
	}
	sort.Slice(repairSuggestions, func(i, j int) bool { return repairSuggestions[j].CreatedAt.Before(repairSuggestions[i].CreatedAt) })
	if len(repairSuggestions) > 20 {
		repairSuggestions = repairSuggestions[:20]
	}
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
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	server, err := h.db.GetServer(c.Request.Context(), serverID)
	if err != nil || server == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	if server.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
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

// Context Config handlers (server id in path; require ownership)
func (h *Handler) GetContextConfigs(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	configs, err := h.db.GetContextConfigs(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

func (h *Handler) CreateContextConfig(c *gin.Context) {
	serverID := c.Param("id")
	if h.requireServerOwnership(c, serverID) == nil {
		return
	}
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

func (h *Handler) DeleteContextConfig(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.DeleteContextConfig(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// requireCompositionOwnership loads the composition and returns it only if the current user owns it.
func (h *Handler) requireCompositionOwnership(c *gin.Context, compositionID string) *models.ServerComposition {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil
	}
	comp, err := h.db.GetComposition(c.Request.Context(), compositionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil
	}
	if comp == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "composition not found"})
		return nil
	}
	if comp.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you do not have access to this composition"})
		return nil
	}
	return comp
}

// Composition handlers (all require auth; list/create scoped to user; get/update/delete/export require ownership)
func (h *Handler) ListCompositions(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	compositions, err := h.db.ListCompositions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, compositions)
}

func (h *Handler) CreateComposition(c *gin.Context) {
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		ServerIDs   []string `json:"server_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	composition, err := h.db.CreateComposition(c.Request.Context(), req.Name, req.Description, req.ServerIDs, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, composition)
}

func (h *Handler) GetComposition(c *gin.Context) {
	id := c.Param("id")
	comp := h.requireCompositionOwnership(c, id)
	if comp == nil {
		return
	}
	c.JSON(http.StatusOK, comp)
}

func (h *Handler) UpdateComposition(c *gin.Context) {
	id := c.Param("id")
	if h.requireCompositionOwnership(c, id) == nil {
		return
	}
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		ServerIDs   []string `json:"server_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	composition, err := h.db.UpdateComposition(c.Request.Context(), id, req.Name, req.Description, req.ServerIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, composition)
}

func (h *Handler) DeleteComposition(c *gin.Context) {
	id := c.Param("id")
	if h.requireCompositionOwnership(c, id) == nil {
		return
	}
	if err := h.db.DeleteComposition(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ExportComposition(c *gin.Context) {
	id := c.Param("id")
	composition := h.requireCompositionOwnership(c, id)
	if composition == nil {
		return
	}
	// Parse options from request body
	var options struct {
		PrefixToolNames bool `json:"prefix_tool_names"`
		MergeResources  bool `json:"merge_resources"`
		MergePrompts    bool `json:"merge_prompts"`
	}
	options.MergeResources = true
	options.MergePrompts = true
	if err := c.ShouldBindJSON(&options); err != nil {
		// Use defaults if no body provided
	}

	// Get all servers in the composition
	servers := make([]*models.Server, 0, len(composition.ServerIDs))
	for _, serverID := range composition.ServerIDs {
		server, err := h.db.GetServer(c.Request.Context(), serverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Server %s not found", serverID)})
			return
		}
		servers = append(servers, server)
	}

	// Generate composition zip
	genOptions := generator.CompositionOptions{
		PrefixToolNames: options.PrefixToolNames,
		MergeResources:  options.MergeResources,
		MergePrompts:    options.MergePrompts,
	}

	zipData, err := h.generator.GenerateCompositionZip(composition, servers, genOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("%s-composition.zip", strings.ReplaceAll(strings.ToLower(composition.Name), " ", "-"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/zip", zipData)
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

// PreviewOpenAPIImport parses OpenAPI spec and returns preview without saving
func (h *Handler) PreviewOpenAPIImport(c *gin.Context) {
	var req struct {
		Spec string `json:"spec" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.openapiParser.Parse([]byte(req.Spec))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse OpenAPI spec: %v", err)})
		return
	}

	// Return preview
	preview := gin.H{
		"server": gin.H{
			"name":        result.ServerName,
			"description": result.ServerDesc,
			"version":     result.ServerVersion,
			"base_url":    result.BaseURL,
		},
		"tools_count": len(result.Tools),
		"tools":       result.Tools,
		"auth":        result.AuthConfig,
	}

	c.JSON(http.StatusOK, preview)
}

// ImportOpenAPI creates a server and tools from OpenAPI spec
func (h *Handler) ImportOpenAPI(c *gin.Context) {
	var req struct {
		Spec        string `json:"spec" binding:"required"`
		ServerName  string `json:"server_name"`  // Optional override
		Description string `json:"description"`  // Optional override
		BaseURL     string `json:"base_url"`     // Optional override
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.openapiParser.Parse([]byte(req.Spec))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse OpenAPI spec: %v", err)})
		return
	}

	// Apply overrides
	if req.ServerName != "" {
		result.ServerName = req.ServerName
	}
	if req.Description != "" {
		result.ServerDesc = req.Description
	}
	if req.BaseURL != "" {
		result.BaseURL = req.BaseURL
	}

	// Generate server ID
	serverID := uuid.New().String()

	// Convert to models
	server, tools := result.ToServerAndTools(serverID)

	// Create server in database (owned by current user)
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	serverReq := models.CreateServerRequest{
		Name:        server.Name,
		Description: server.Description,
		Version:     server.Version,
		OwnerID:     userID,
	}
	createdServer, err := h.db.CreateServer(c.Request.Context(), serverReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create server: %v", err)})
		return
	}

	// Create tools in database
	createdTools := []models.Tool{}
	for _, tool := range tools {
		tool.ServerID = createdServer.ID

		toolReq := models.CreateToolRequest{
			ServerID:        createdServer.ID,
			Name:            tool.Name,
			Description:     tool.Description,
			ExecutionType:   tool.ExecutionType,
			InputSchema:     tool.InputSchema,
			OutputSchema:    tool.OutputSchema,
			ExecutionConfig: tool.ExecutionConfig,
		}
		createdTool, err := h.db.CreateTool(c.Request.Context(), toolReq)
		if err != nil {
			// Log error but continue with other tools
			fmt.Printf("Warning: Failed to create tool %s: %v\n", tool.Name, err)
			continue
		}
		createdTools = append(createdTools, *createdTool)
	}

	c.JSON(http.StatusCreated, gin.H{
		"server":        createdServer,
		"tools_created": len(createdTools),
		"tools":         createdTools,
	})
}

// Flow handlers

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

// GetSecurityScore returns the server's security score based on the SlowMist MCP Security Checklist.
func (h *Handler) GetSecurityScore(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	policiesByTool := make(map[string][]models.Policy)
	for _, t := range server.Tools {
		policies, err := h.db.GetPoliciesByTool(c.Request.Context(), t.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		policiesByTool[t.ID] = policies
	}
	result := security.Score(server, policiesByTool)
	c.JSON(http.StatusOK, result)
}

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

	// Parse nodes and edges
	var nodes []models.FlowNode
	var edges []models.FlowEdge
	
	if err := json.Unmarshal(flow.Nodes, &nodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid nodes format"})
		return
	}
	if err := json.Unmarshal(flow.Edges, &edges); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid edges format"})
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

	// Parse nodes and edges
	var nodes []models.FlowNode
	var edges []models.FlowEdge
	if err := json.Unmarshal(flow.Nodes, &nodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid nodes format"})
		return
	}
	if err := json.Unmarshal(flow.Edges, &edges); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid edges format"})
		return
	}

	// Build input schema from trigger node
	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	// Extract input schema from trigger node if defined
	for _, node := range nodes {
		if node.Type == "trigger" {
			var data map[string]interface{}
			json.Unmarshal(node.Data, &data)
			if schema, ok := data["inputSchema"]; ok {
				inputSchema = schema.(map[string]interface{})
			}
			break
		}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tool":    tool,
		"message": "Flow converted to tool successfully",
	})
}

// Auth handlers

// Register creates a user account (email + name only). The client must then complete passkey registration via WebAuthn register/begin and register/finish to sign in.
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	existingUser, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	user, err := h.db.CreateUser(c.Request.Context(), req.Email, req.Name, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Account created. Use passkey registration (webauthn/register/begin then register/finish) to add a passkey and sign in.",
		"user":    user,
	})
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	user, err := h.db.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	})
}

// WebAuthnRegisterBegin starts passkey registration: returns creation options and session_id.
func (h *Handler) WebAuthnRegisterBegin(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
		return
	}
	user, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found; register first with email and name"})
		return
	}
	rows, err := h.db.GetWebAuthnCredentials(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load credentials"})
		return
	}
	credData := make([][]byte, 0, len(rows))
	for _, r := range rows {
		credData = append(credData, r.Data)
	}
	waUser, err := webauthnpkg.NewWebAuthnUser(user, credData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	creation, session, err := h.webauthn.BeginRegistration(waUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "begin registration: " + err.Error()})
		return
	}
	sessionID := uuid.New().String()
	h.sessionStore.Set(sessionID, session)
	// Return options in the format the frontend expects (CredentialCreationOptions)
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"options":    creation,
	})
}

// WebAuthnRegisterFinish completes passkey registration and returns JWT.
func (h *Handler) WebAuthnRegisterFinish(c *gin.Context) {
	var req struct {
		SessionID string          `json:"session_id" binding:"required"`
		Response  json.RawMessage `json:"response" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and response required"})
		return
	}
	session := h.sessionStore.Get(req.SessionID)
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired session"})
		return
	}
	userID := string(session.UserID)
	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	rows, _ := h.db.GetWebAuthnCredentials(c.Request.Context(), user.ID)
	credData := make([][]byte, 0, len(rows))
	for _, r := range rows {
		credData = append(credData, r.Data)
	}
	waUser, _ := webauthnpkg.NewWebAuthnUser(user, credData)
	// Build a request that has the response in the body so the library can parse it
	r := c.Request
	r.Body = io.NopCloser(bytes.NewReader(req.Response))
	r.ContentLength = int64(len(req.Response))
	credential, err := h.webauthn.FinishRegistration(waUser, *session, r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification failed: " + err.Error()})
		return
	}
	credJSON, _ := json.Marshal(credential)
	if err := h.db.SaveWebAuthnCredential(c.Request.Context(), user.ID, credential.ID, credJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save credential"})
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	c.JSON(http.StatusOK, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// WebAuthnLoginBegin starts passkey authentication: returns assertion options and session_id.
func (h *Handler) WebAuthnLoginBegin(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
		return
	}
	user, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no account for this email"})
		return
	}
	rows, err := h.db.GetWebAuthnCredentials(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load credentials"})
		return
	}
	credData := make([][]byte, 0, len(rows))
	for _, r := range rows {
		credData = append(credData, r.Data)
	}
	waUser, err := webauthnpkg.NewWebAuthnUser(user, credData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	assertion, session, err := h.webauthn.BeginLogin(waUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "begin login: " + err.Error()})
		return
	}
	sessionID := uuid.New().String()
	h.sessionStore.Set(sessionID, session)
	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"options":    assertion,
	})
}

// WebAuthnLoginFinish completes passkey authentication and returns JWT.
func (h *Handler) WebAuthnLoginFinish(c *gin.Context) {
	var req struct {
		SessionID string          `json:"session_id" binding:"required"`
		Response  json.RawMessage `json:"response" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and response required"})
		return
	}
	session := h.sessionStore.Get(req.SessionID)
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired session"})
		return
	}
	userID := string(session.UserID)
	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	rows, _ := h.db.GetWebAuthnCredentials(c.Request.Context(), user.ID)
	credData := make([][]byte, 0, len(rows))
	for _, r := range rows {
		credData = append(credData, r.Data)
	}
	waUser, _ := webauthnpkg.NewWebAuthnUser(user, credData)
	r := c.Request
	r.Body = io.NopCloser(bytes.NewReader(req.Response))
	r.ContentLength = int64(len(req.Response))
	_, err = h.webauthn.FinishLogin(waUser, *session, r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification failed: " + err.Error()})
		return
	}
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	c.JSON(http.StatusOK, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// AuthMiddleware validates JWT token
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		// Prefer custom user_id claim; fall back to standard "sub" (subject) for compatibility
		userID := strings.TrimSpace(claims.UserID)
		if userID == "" && claims.Subject != "" {
			userID = strings.TrimSpace(claims.Subject)
		}
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: missing user identity"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("userEmail", claims.Email)
		c.Next()
	}
}

