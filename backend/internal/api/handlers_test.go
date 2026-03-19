package api

import (
	"encoding/json"
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
		})
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
		{input: "auto_flow", want: hostedAuthModeBearerToken},       // legacy mapping
		{input: "caller_identity", want: hostedAuthModeNoAuth},      // legacy mapping
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
