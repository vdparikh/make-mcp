package webauthn

import (
	"net/url"
	"os"
	"strings"

	wa "github.com/go-webauthn/webauthn/webauthn"
)

// NewWebAuthn creates a WebAuthn instance for passkey registration and authentication.
func NewWebAuthn() (*wa.WebAuthn, error) {
	rpID := os.Getenv("WEBAUTHN_RP_ID")
	if rpID == "" {
		rpID = "localhost"
	}
	originsStr := os.Getenv("WEBAUTHN_RP_ORIGINS")
	origins := []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:3000", "http://127.0.0.1:5173"}
	if originsStr != "" {
		origins = strings.Split(originsStr, ",")
		for i, o := range origins {
			origins[i] = strings.TrimSpace(o)
		}
	}
	for i, o := range origins {
		if o != "" && !strings.HasPrefix(o, "http") {
			origins[i] = "https://" + o
		}
	}

	config := &wa.Config{
		RPID:          rpID,
		RPDisplayName: "Make MCP",
		RPOrigins:     origins,
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
