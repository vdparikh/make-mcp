// Package hostedsecurity resolves hosted MCP access policies: per-environment profiles,
// IP allowlists, OIDC bearer tokens, and optional mTLS client-certificate hints from the edge.
package hostedsecurity

import (
	"encoding/json"
	"strings"
)

// Config is stored in servers.hosted_security_config (JSONB).
type Config struct {
	// DefaultEnv is used when the client omits X-Make-MCP-Env (e.g. "prod", "dev").
	DefaultEnv string `json:"default_env,omitempty"`
	// EnvProfiles maps environment name -> overrides merged onto the server row.
	EnvProfiles map[string]EnvProfile `json:"env_profiles,omitempty"`
	// OIDC default when an env profile does not set oidc (mode must be oidc).
	OIDC *OIDCConfig `json:"oidc,omitempty"`
	// MTLS default trusted cert fingerprints when env profile does not set mtls.
	MTLS *MTLSConfig `json:"mtls,omitempty"`
	// OAuthBFF optional: Make MCP acts as OAuth facade (authorize/callback) to upstream IdP (e.g. Keycloak).
	OAuthBFF *OAuthBFFConfig `json:"oauth_bff,omitempty"`
}

// EnvProfile overrides hosted auth for one deployment environment.
type EnvProfile struct {
	IPAllowlist []string `json:"ip_allowlist,omitempty"`
	// HostedAuthMode overrides servers.hosted_auth_mode when non-empty:
	// no_auth, bearer_token, oidc, mtls
	HostedAuthMode string `json:"hosted_auth_mode,omitempty"`
	// RequireCallerIdentity overrides servers.require_caller_identity when non-nil.
	RequireCallerIdentity *bool `json:"require_caller_identity,omitempty"`
	// RequireMTLS when true requires a matching X-Make-MCP-Client-Cert-SHA256 in addition to bearer/OIDC.
	RequireMTLS bool `json:"require_mtls,omitempty"`
	OIDC        *OIDCConfig `json:"oidc,omitempty"`
	MTLS        *MTLSConfig `json:"mtls,omitempty"`
	OAuthBFF    *OAuthBFFConfig `json:"oauth_bff,omitempty"`
}

// OAuthBFFConfig enables server-side OAuth (BFF): browser redirects to upstream IdP, code exchange on Make MCP, then redirect back to MCP client with tokens.
// Client secret must not be stored in JSON; use ClientSecretEnv pointing to an environment variable name.
type OAuthBFFConfig struct {
	Enabled bool `json:"enabled"`
	// UpstreamIssuer is the IdP realm issuer (e.g. https://idp.example/realms/corp).
	UpstreamIssuer string `json:"upstream_issuer"`
	// ClientID is the confidential OAuth client registered at the upstream IdP; redirect URI must include Make MCP /api/oauth/callback.
	ClientID string `json:"client_id"`
	// ClientSecretEnv names an environment variable holding the client secret (e.g. KEYCLOAK_OAUTH_CLIENT_SECRET).
	ClientSecretEnv string `json:"client_secret_env,omitempty"`
}

// OIDCConfig validates Authorization: Bearer <JWT> from an OIDC provider (resource-server style).
type OIDCConfig struct {
	Issuer   string `json:"issuer"`
	Audience string `json:"audience"`
	// JWKSURL optional; if empty, discovered from issuer /.well-known/openid-configuration
	JWKSURL string `json:"jwks_url,omitempty"`
}

// MTLSConfig lists trusted client certificate SHA-256 fingerprints (hex, lowercase),
// as observed at the ingress and forwarded via X-Make-MCP-Client-Cert-SHA256.
type MTLSConfig struct {
	TrustedClientCertSHA256 []string `json:"trusted_client_cert_sha256,omitempty"`
}

// ParseConfig unmarshals JSON; nil or empty is valid (no extra restrictions).
func ParseConfig(raw []byte) (*Config, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || string(raw) == "null" {
		return &Config{}, nil
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ResolvedProfile is the effective policy for one request after merging server + env profile.
type ResolvedProfile struct {
	EnvName                 string
	HostedAuthMode          string
	RequireCallerIdentity   bool
	IPAllowlist             []string
	RequireMTLS             bool
	OIDC                    *OIDCConfig
	MTLS                    *MTLSConfig
	TrustedCertFingerprints []string // merged: env profile mtls + RequireMTLS path
	// OAuthBFF when set and enabled: serve /.well-known OAuth metadata and /api/oauth/* facade to UpstreamIssuer.
	OAuthBFF *OAuthBFFConfig
}
