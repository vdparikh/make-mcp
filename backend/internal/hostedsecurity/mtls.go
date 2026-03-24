package hostedsecurity

import (
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// HeaderClientCertSHA256 is set by the ingress / API gateway with the client cert fingerprint (hex, lowercase).
const HeaderClientCertSHA256 = "X-Make-MCP-Client-Cert-SHA256"

// FingerprintAllowed returns true if header matches one of trusted SHA-256 hex digests.
func FingerprintAllowed(header string, trusted []string) bool {
	h := normalizeHexFP(header)
	if h == "" || len(trusted) == 0 {
		return false
	}
	hb, err := hex.DecodeString(h)
	if err != nil || len(hb) != 32 {
		return false
	}
	for _, t := range trusted {
		tb := normalizeHexFP(t)
		if tb == "" {
			continue
		}
		db, err := hex.DecodeString(tb)
		if err != nil || len(db) != 32 {
			continue
		}
		if subtle.ConstantTimeCompare(hb, db) == 1 {
			return true
		}
	}
	return false
}

func normalizeHexFP(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}
