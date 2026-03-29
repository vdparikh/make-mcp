package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

func newTestContext(method string) *gin.Context {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, "/api/users/u/s", nil)
	c.Request = req
	return c
}

func TestHostedAccessTokenFromRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name: "authorization bearer takes precedence",
			headers: map[string]string{
				"Authorization":    "Bearer token-1",
				hostedAccessHeader: "token-2",
			},
			want: "token-1",
		},
		{
			name: "accepts custom hosted header",
			headers: map[string]string{
				hostedAccessHeader: "token-2",
			},
			want: "token-2",
		},
		{
			name: "accepts legacy api key header",
			headers: map[string]string{
				"X-MCP-API-Key": "token-3",
			},
			want: "token-3",
		},
		{
			name: "invalid authorization format ignored",
			headers: map[string]string{
				"Authorization": "Basic abc",
			},
			want: "",
		},
		{
			name:    "missing headers returns empty",
			headers: map[string]string{},
			want:    "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newTestContext("GET")
			for k, v := range tc.headers {
				c.Request.Header.Set(k, v)
			}
			got := hostedAccessTokenFromRequest(c)
			if got != tc.want {
				t.Fatalf("hostedAccessTokenFromRequest() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSecureStringEquals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "equal strings", a: "abc123", b: "abc123", want: true},
		{name: "different values", a: "abc123", b: "abc124", want: false},
		{name: "different lengths", a: "abc", b: "abcd", want: false},
		{name: "empty values", a: "", b: "", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := secureStringEquals(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("secureStringEquals(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestBuildHostedMCPConfig(t *testing.T) {
	t.Parallel()
	h := &Handler{}

	cases := []struct {
		name                  string
		authMode              string
		requireCallerIdentity bool
		accessKey             string
		wantAuth              bool
		wantCallerHeaders     bool
	}{
		{
			name:                  "bearer_token includes Authorization header",
			authMode:              hostedAuthModeBearerToken,
			requireCallerIdentity: false,
			accessKey:             "secret-token",
			wantAuth:              true,
			wantCallerHeaders:     false,
		},
		{
			name:                  "no_auth omits Authorization header",
			authMode:              hostedAuthModeNoAuth,
			requireCallerIdentity: false,
			accessKey:             "secret-token",
			wantAuth:              false,
			wantCallerHeaders:     false,
		},
		{
			name:                  "caller_identity toggle adds caller headers",
			authMode:              hostedAuthModeNoAuth,
			requireCallerIdentity: true,
			accessKey:             "secret-token",
			wantAuth:              false,
			wantCallerHeaders:     true,
		},
		{
			name:                  "bearer_token + caller_identity adds both headers",
			authMode:              hostedAuthModeBearerToken,
			requireCallerIdentity: true,
			accessKey:             "secret-token",
			wantAuth:              true,
			wantCallerHeaders:     true,
		},
		{
			name:                  "no_auth without caller identity has no headers",
			authMode:              hostedAuthModeNoAuth,
			requireCallerIdentity: false,
			accessKey:             "secret-token",
			wantAuth:              false,
			wantCallerHeaders:     false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := &models.Server{
				HostedAuthMode:        tc.authMode,
				RequireCallerIdentity: tc.requireCallerIdentity,
			}
			cfg, err := h.buildHostedMCPConfig("demo", "https://example.test/api/users/u/demo", server, tc.accessKey)
			if err != nil {
				t.Fatalf("buildHostedMCPConfig() error = %v", err)
			}
			var parsed map[string]map[string]map[string]interface{}
			if err := json.Unmarshal(cfg, &parsed); err != nil {
				t.Fatalf("unmarshal config: %v", err)
			}
			demo := parsed["mcpServers"]["demo"]
			headers, _ := demo["headers"].(map[string]interface{})

			if tc.wantAuth {
				if headers == nil {
					t.Fatal("expected headers to be present")
				}
				if got, _ := headers["Authorization"].(string); got != "Bearer "+tc.accessKey {
					t.Fatalf("Authorization = %q, want %q", got, "Bearer "+tc.accessKey)
				}
			} else if headers != nil {
				if _, ok := headers["Authorization"]; ok {
					t.Fatal("Authorization header should not be present")
				}
			}

			if tc.wantCallerHeaders {
				if headers == nil {
					t.Fatal("expected headers for caller identity")
				}
				if got, _ := headers["X-Make-MCP-Caller-Id"].(string); got == "" {
					t.Fatal("expected X-Make-MCP-Caller-Id placeholder")
				}
			} else if headers != nil {
				if _, ok := headers["X-Make-MCP-Caller-Id"]; ok {
					t.Fatal("X-Make-MCP-Caller-Id should not be present")
				}
			}
			if headers != nil {
				if _, ok := headers["X-Make-MCP-Tenant-Id"]; ok {
					t.Fatal("X-Make-MCP-Tenant-Id should not be present in generated MCP config")
				}
			}
		})
	}
}

func TestBuildHostedMCPConfigOAuthBFF(t *testing.T) {
	t.Parallel()
	h := &Handler{}
	sec := `{
		"default_env": "dev",
		"oidc": { "issuer": "http://localhost:8180/realms/r" },
		"oauth_bff": {
			"enabled": true,
			"upstream_issuer": "http://localhost:8180/realms/r",
			"client_id": "make-mcp-bff"
		}
	}`
	server := &models.Server{
		HostedAuthMode:       hostedAuthModeOIDC,
		HostedSecurityConfig: json.RawMessage(sec),
	}
	cfg, err := h.buildHostedMCPConfig("demo", "https://example.test/api/users/u/demo", server, "ignored-key")
	if err != nil {
		t.Fatalf("buildHostedMCPConfig: %v", err)
	}
	var parsed map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	demo := parsed["mcpServers"]["demo"]
	auth, _ := demo["auth"].(map[string]interface{})
	if auth == nil {
		t.Fatal("expected auth block for OAuth BFF + OIDC")
	}
	if got, _ := auth["CLIENT_ID"].(string); got != "make-mcp-bff" {
		t.Fatalf("CLIENT_ID = %q, want make-mcp-bff", got)
	}
	scopes, _ := auth["scopes"].([]interface{})
	if len(scopes) < 2 {
		t.Fatalf("expected scopes list, got %v", auth["scopes"])
	}
}

func TestNormalizeHostedAuthMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "", want: hostedAuthModeNoAuth},
		{input: "bearer_token", want: hostedAuthModeBearerToken},
		{input: "no_auth", want: hostedAuthModeNoAuth},
		{input: "auto_flow", want: hostedAuthModeBearerToken},  // legacy mapping
		{input: "caller_identity", want: hostedAuthModeNoAuth}, // legacy mapping
		{input: "oidc", want: hostedAuthModeOIDC},
		{input: "mtls", want: hostedAuthModeMTLS},
		{input: "invalid_mode", want: "", wantErr: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run("input="+tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeHostedAuthMode(tc.input)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeHostedAuthMode(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExecuteGraphQL(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{"widget": map[string]interface{}{"id": "1"}},
		})
	}))
	t.Cleanup(srv.Close)

	cfg, err := json.Marshal(map[string]interface{}{
		"url":     srv.URL,
		"query":   "query { widget { id } }",
		"headers": map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	tool := &models.Tool{
		ExecutionType:   models.ExecutionTypeGraphQL,
		ExecutionConfig: cfg,
	}
	h := &Handler{}
	out, code, err := h.executeTool(context.Background(), tool, map[string]interface{}{"x": 1})
	if err != nil {
		t.Fatalf("executeTool: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	m, ok := out.(map[string]interface{})
	if !ok {
		t.Fatalf("out type %T, want map", out)
	}
	widget, ok := m["widget"].(map[string]interface{})
	if !ok || widget["id"] != "1" {
		t.Fatalf("unexpected data: %#v", out)
	}
}

func TestExecuteGraphQLErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{{"message": "bad field"}},
		})
	}))
	t.Cleanup(srv.Close)

	cfg, _ := json.Marshal(map[string]interface{}{
		"url":   srv.URL,
		"query": "query { x }",
	})
	tool := &models.Tool{ExecutionType: models.ExecutionTypeGraphQL, ExecutionConfig: cfg}
	h := &Handler{}
	_, code, err := h.executeTool(context.Background(), tool, nil)
	if err == nil {
		t.Fatal("expected error from graphql errors array")
	}
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (graphql errors often use HTTP 200)", code)
	}
}

func TestExecuteDatabaseLearningSample(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]interface{}{
		"connection_string": blueprintLearningDBURI,
		"query":             "SELECT * FROM t WHERE id = {{product_id}}",
	})
	tool := &models.Tool{ExecutionType: models.ExecutionTypeDatabase, ExecutionConfig: cfg}
	h := &Handler{}
	out, code, err := h.executeTool(context.Background(), tool, map[string]interface{}{"product_id": "99"})
	if err != nil {
		t.Fatal(err)
	}
	if code != 200 {
		t.Fatalf("code = %d", code)
	}
	m := out.(map[string]interface{})
	if m["test_playground"] != "learning_sample" {
		t.Fatalf("test_playground = %v", m["test_playground"])
	}
	if m["resolved_query"] != "SELECT * FROM t WHERE id = 99" {
		t.Fatalf("resolved_query = %q", m["resolved_query"])
	}
}

func TestExecuteCLIPreview(t *testing.T) {
	t.Parallel()
	h := &Handler{}
	cases := []struct {
		name    string
		command string
		allowed []string
		wantErr bool
		want400 bool
	}{
		{name: "echo allowed", command: "echo {{message}}", allowed: []string{"echo"}, wantErr: false},
		{name: "echo not in allowlist", command: "echo {{message}}", allowed: []string{"ls"}, wantErr: true, want400: true},
		{name: "no allowlist ok", command: "echo {{message}}", allowed: nil, wantErr: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg, _ := json.Marshal(map[string]interface{}{
				"command":          tc.command,
				"allowed_commands": tc.allowed,
				"shell":            "/bin/bash",
				"timeout":          1000,
				"working_dir":      ".",
				"env":              map[string]string{},
			})
			tool := &models.Tool{ExecutionType: models.ExecutionTypeCLI, ExecutionConfig: cfg}
			out, code, err := h.executeTool(context.Background(), tool, map[string]interface{}{"message": "hi"})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tc.want400 && code != 400 {
					t.Fatalf("code = %d, want 400", code)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			m := out.(map[string]interface{})
			if m["resolved_command"] != "echo hi" {
				t.Fatalf("resolved_command = %q", m["resolved_command"])
			}
		})
	}
}

func TestExecuteInProcessCodePreview(t *testing.T) {
	t.Parallel()
	cfg, _ := json.Marshal(map[string]interface{}{
		"runtime": "node",
		"snippet": "return {};",
	})
	tool := &models.Tool{
		ExecutionType:   models.ExecutionTypeJavaScript,
		ExecutionConfig: cfg,
	}
	h := &Handler{}
	out, code, err := h.executeTool(context.Background(), tool, map[string]interface{}{"a": 1})
	if err != nil || code != 200 {
		t.Fatalf("err=%v code=%d", err, code)
	}
	m := out.(map[string]interface{})
	if m["test_playground"] != "simulation" {
		t.Fatalf("got %#v", m)
	}
	gotET, ok := m["execution_type"].(models.ExecutionType)
	if !ok || gotET != models.ExecutionTypeJavaScript {
		t.Fatalf("execution_type = %v (%T), want javascript", m["execution_type"], m["execution_type"])
	}
}

func TestBuildPolicyEvalContext(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input map[string]interface{}
		wantU string
		wantO string
		wantR []string
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name: "maps user org and roles",
			input: map[string]interface{}{
				"user_id":         "u1",
				"organization_id": "org1",
				"roles":           []interface{}{"admin", "operator"},
			},
			wantU: "u1",
			wantO: "org1",
			wantR: []string{"admin", "operator"},
		},
		{
			name: "ignores invalid role entries",
			input: map[string]interface{}{
				"roles": []interface{}{"reader", 123, true},
			},
			wantR: []string{"reader"},
		},
		{
			name: "ignores wrong field types",
			input: map[string]interface{}{
				"user_id":         42,
				"organization_id": false,
				"roles":           "admin",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildPolicyEvalContext(tc.input)
			if got == nil {
				t.Fatal("got nil context")
			}
			if got.Custom == nil {
				t.Fatal("expected non-nil custom map")
			}
			if got.UserID != tc.wantU {
				t.Fatalf("UserID = %q, want %q", got.UserID, tc.wantU)
			}
			if got.OrganizationID != tc.wantO {
				t.Fatalf("OrganizationID = %q, want %q", got.OrganizationID, tc.wantO)
			}
			if len(got.Roles) != len(tc.wantR) {
				t.Fatalf("Roles len = %d, want %d (%v)", len(got.Roles), len(tc.wantR), got.Roles)
			}
			for i := range tc.wantR {
				if got.Roles[i] != tc.wantR[i] {
					t.Fatalf("Roles[%d] = %q, want %q", i, got.Roles[i], tc.wantR[i])
				}
			}
		})
	}
}

func TestExtractTriggerInputSchema(t *testing.T) {
	t.Parallel()

	triggerData, _ := json.Marshal(map[string]interface{}{
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{"type": "string"},
			},
		},
	})
	badTriggerData, _ := json.Marshal(map[string]interface{}{
		"inputSchema": "not-an-object",
	})

	cases := []struct {
		name  string
		nodes []models.FlowNode
		want  bool
	}{
		{
			name: "returns schema from trigger",
			nodes: []models.FlowNode{
				{Type: "api"},
				{Type: "trigger", Data: triggerData},
			},
			want: true,
		},
		{
			name: "returns nil when no trigger",
			nodes: []models.FlowNode{
				{Type: "api"},
			},
			want: false,
		},
		{
			name: "returns nil on invalid schema type",
			nodes: []models.FlowNode{
				{Type: "trigger", Data: badTriggerData},
			},
			want: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractTriggerInputSchema(tc.nodes)
			if tc.want && got == nil {
				t.Fatal("expected schema, got nil")
			}
			if !tc.want && got != nil {
				t.Fatalf("expected nil, got %#v", got)
			}
		})
	}
}

func TestParseFlowGraph(t *testing.T) {
	t.Parallel()

	goodNodes, _ := json.Marshal([]map[string]interface{}{
		{"id": "n1", "type": "trigger", "position": map[string]float64{"x": 0, "y": 0}, "data": map[string]interface{}{}},
	})
	goodEdges, _ := json.Marshal([]map[string]interface{}{
		{"id": "e1", "source": "n1", "target": "n2"},
	})

	cases := []struct {
		name    string
		flow    *models.Flow
		wantErr string
	}{
		{
			name: "valid graph",
			flow: &models.Flow{Nodes: goodNodes, Edges: goodEdges},
		},
		{
			name:    "invalid nodes",
			flow:    &models.Flow{Nodes: json.RawMessage(`{`), Edges: goodEdges},
			wantErr: "invalid nodes format",
		},
		{
			name:    "invalid edges",
			flow:    &models.Flow{Nodes: goodNodes, Edges: json.RawMessage(`{`)},
			wantErr: "invalid edges format",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			nodes, edges, err := parseFlowGraph(tc.flow)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(nodes) == 0 {
					t.Fatal("expected parsed nodes")
				}
				if len(edges) == 0 {
					t.Fatal("expected parsed edges")
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tc.wantErr)
			}
		})
	}
}
