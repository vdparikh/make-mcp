package webauthn

import (
	"fmt"
	"net/url"
	"strings"

	wa "github.com/go-webauthn/webauthn/webauthn"
)

// NewWebAuthn creates a WebAuthn instance from explicit RP ID and origins (from application config).
func NewWebAuthn(rpID string, origins []string) (*wa.WebAuthn, error) {
	rpID = strings.TrimSpace(rpID)
	if rpID == "" {
		return nil, fmt.Errorf("webauthn: rp_id is required")
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("webauthn: at least one origin is required")
	}
	norm := make([]string, len(origins))
	for i, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" && !strings.HasPrefix(o, "http") {
			o = "https://" + o
		}
		norm[i] = o
	}

	config := &wa.Config{
		RPID:          rpID,
		RPDisplayName: "Make MCP",
		RPOrigins:     norm,
	}

	return wa.New(config)
}

// ValidateOrigin returns true if the given origin is allowed.
func ValidateOrigin(allowed []string, requestOrigin string) bool {
	u, err := url.Parse(requestOrigin)
	if err != nil {
		return false
	}
	origin := u.Scheme + "://" + u.Host
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}
