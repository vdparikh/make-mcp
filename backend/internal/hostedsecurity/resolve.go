package hostedsecurity

import (
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

const HeaderEnv = "X-Make-MCP-Env"

// Resolve merges server-level settings with hosted_security_config for the requested environment.
func Resolve(server *models.Server, secRaw []byte, envHeader string) (*ResolvedProfile, error) {
	cfg, err := ParseConfig(secRaw)
	if err != nil {
		return nil, err
	}
	env := strings.TrimSpace(strings.ToLower(envHeader))
	if env == "" {
		env = strings.TrimSpace(strings.ToLower(cfg.DefaultEnv))
	}
	if env == "" {
		env = "prod"
	}

	mode := strings.TrimSpace(strings.ToLower(server.HostedAuthMode))
	if mode == "" {
		mode = "no_auth"
	}
	requireCaller := server.RequireCallerIdentity

	var ipList []string
	var oidc *OIDCConfig
	var mtls *MTLSConfig
	requireMTLS := false
	var trusted []string
	var oauthBFF *OAuthBFFConfig

	if cfg.EnvProfiles != nil {
		if ep, ok := cfg.EnvProfiles[env]; ok {
			if len(ep.IPAllowlist) > 0 {
				ipList = append([]string(nil), ep.IPAllowlist...)
			}
			if strings.TrimSpace(ep.HostedAuthMode) != "" {
				mode = strings.TrimSpace(strings.ToLower(ep.HostedAuthMode))
			}
			if ep.RequireCallerIdentity != nil {
				requireCaller = *ep.RequireCallerIdentity
			}
			requireMTLS = ep.RequireMTLS
			if ep.OIDC != nil && strings.TrimSpace(ep.OIDC.Issuer) != "" {
				oidc = ep.OIDC
			}
			if ep.MTLS != nil && len(ep.MTLS.TrustedClientCertSHA256) > 0 {
				mtls = ep.MTLS
				trusted = append(trusted, normalizeFPList(ep.MTLS.TrustedClientCertSHA256)...)
			}
			if ep.OAuthBFF != nil && ep.OAuthBFF.Enabled && strings.TrimSpace(ep.OAuthBFF.UpstreamIssuer) != "" && strings.TrimSpace(ep.OAuthBFF.ClientID) != "" {
				oauthBFF = ep.OAuthBFF
			}
		}
	}
	if oidc == nil && cfg.OIDC != nil && strings.TrimSpace(cfg.OIDC.Issuer) != "" {
		oidc = cfg.OIDC
	}
	if len(trusted) == 0 && cfg.MTLS != nil && len(cfg.MTLS.TrustedClientCertSHA256) > 0 {
		mtls = cfg.MTLS
		trusted = append(trusted, normalizeFPList(cfg.MTLS.TrustedClientCertSHA256)...)
	}
	if oauthBFF == nil && cfg.OAuthBFF != nil && cfg.OAuthBFF.Enabled && strings.TrimSpace(cfg.OAuthBFF.UpstreamIssuer) != "" && strings.TrimSpace(cfg.OAuthBFF.ClientID) != "" {
		oauthBFF = cfg.OAuthBFF
	}

	return &ResolvedProfile{
		EnvName:               env,
		HostedAuthMode:        mode,
		RequireCallerIdentity: requireCaller,
		IPAllowlist:           ipList,
		RequireMTLS:           requireMTLS,
		OIDC:                  oidc,
		MTLS:                  mtls,
		TrustedCertFingerprints: dedupeFP(trusted),
		OAuthBFF:                oauthBFF,
	}, nil
}

func normalizeFPList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		t := strings.TrimSpace(strings.ToLower(s))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func dedupeFP(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
