package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	auditHostedAccessKeyRotated   = "hosted_access_key_rotated"
	auditHostedSecurityConfigSave = "hosted_security_config_updated"
)

// GetHostedSecurity returns non-secret hosted security settings for a server.
func (h *Handler) GetHostedSecurity(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	hasKey := strings.TrimSpace(server.HostedAccessKey) != ""
	c.JSON(http.StatusOK, gin.H{
		"hosted_auth_mode":          server.HostedAuthMode,
		"require_caller_identity":   server.RequireCallerIdentity,
		"hosted_security_config":    json.RawMessage(server.HostedSecurityConfig),
		"hosted_runtime_config":     json.RawMessage(server.HostedRuntimeConfig),
		"has_hosted_access_key":     hasKey,
		"env_header":                "X-Make-MCP-Env",
		"client_cert_header":        "X-Make-MCP-Client-Cert-SHA256",
	})
}

type putHostedSecurityRequest struct {
	HostedAuthMode          string          `json:"hosted_auth_mode,omitempty"`
	RequireCallerIdentity   *bool           `json:"require_caller_identity,omitempty"`
	HostedSecurityConfig    json.RawMessage `json:"hosted_security_config,omitempty"`
}

// PutHostedSecurity updates hosted auth mode, caller requirement, and/or security JSON.
func (h *Handler) PutHostedSecurity(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	actor := h.currentUserID(c)
	var req putHostedSecurityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.HostedSecurityConfig) > 256*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hosted_security_config too large"})
		return
	}
	if strings.TrimSpace(req.HostedAuthMode) != "" {
		mode, err := normalizeHostedAuthMode(req.HostedAuthMode)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.db.UpdateServerHostedAuthMode(c.Request.Context(), id, mode); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update hosted auth mode"})
			return
		}
	}
	if req.RequireCallerIdentity != nil {
		if err := h.db.UpdateServerRequireCallerIdentity(c.Request.Context(), id, *req.RequireCallerIdentity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update caller identity requirement"})
			return
		}
	}
	if len(req.HostedSecurityConfig) > 0 {
		if err := h.db.UpdateServerHostedSecurityConfig(c.Request.Context(), id, req.HostedSecurityConfig); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save hosted security config"})
			return
		}
		meta, _ := json.Marshal(map[string]string{"action": auditHostedSecurityConfigSave})
		_ = h.db.InsertHostedSecurityAudit(c.Request.Context(), id, actor, auditHostedSecurityConfigSave, "server", id, meta)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RotateHostedAccessKey rotates the static hosted bearer/API key and returns the new secret once.
func (h *Handler) RotateHostedAccessKey(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	actor := h.currentUserID(c)
	newKey, err := h.db.RotateServerHostedAccessKey(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate hosted access key"})
		return
	}
	meta, _ := json.Marshal(map[string]string{"rotated_at": time.Now().UTC().Format(time.RFC3339)})
	_ = h.db.InsertHostedSecurityAudit(c.Request.Context(), id, actor, auditHostedAccessKeyRotated, "hosted_access_key", id, meta)
	c.JSON(http.StatusOK, gin.H{
		"hosted_access_key": newKey,
		"warning":           "store this key securely; it is not shown again",
	})
}

// ListHostedSecurityAudit returns recent security audit events (JSON).
func (h *Handler) ListHostedSecurityAudit(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	owner := h.currentUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	rows, err := h.db.ListHostedSecurityAudit(c.Request.Context(), id, owner, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": rows})
}

// ExportHostedSecurityAudit returns audit rows as CSV for compliance exports.
func (h *Handler) ExportHostedSecurityAudit(c *gin.Context) {
	id := c.Param("id")
	if h.requireServerOwnership(c, id) == nil {
		return
	}
	owner := h.currentUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	rows, err := h.db.ListHostedSecurityAudit(c.Request.Context(), id, owner, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="hosted-security-audit-%s.csv"`, id))
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"id", "server_id", "actor_user_id", "action", "resource_type", "resource_id", "metadata", "created_at"})
	for _, e := range rows {
		meta := string(e.Metadata)
		if meta == "" {
			meta = "{}"
		}
		_ = w.Write([]string{
			e.ID, e.ServerID, e.ActorUserID, e.Action, e.ResourceType, e.ResourceID,
			meta, e.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	w.Flush()
}
