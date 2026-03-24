package hostedruntime

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// EgressPolicy controls outbound HTTP(S) from generated tool code (fetch), not OS-level firewall.
// npm install during container start still uses full network.
const (
	EgressAllowAll    = "allow_all"
	EgressDenyDefault = "deny_default"
)

// NormalizeHostRule trims and lowercases a hostname or wildcard rule like *.example.com.
func NormalizeHostRule(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// HostAllowed returns true if hostname matches any rule (exact or *.domain suffix).
func HostAllowed(host string, rules []string) bool {
	h := NormalizeHostRule(host)
	if h == "" {
		return false
	}
	for _, raw := range rules {
		r := NormalizeHostRule(raw)
		if r == "" {
			continue
		}
		if strings.HasPrefix(r, "*.") {
			domain := strings.TrimPrefix(r, "*.")
			// *.example.com matches foo.example.com but not example.com (apex must be listed explicitly).
			if strings.HasSuffix(h, "."+domain) {
				return true
			}
			continue
		}
		if h == r {
			return true
		}
	}
	return false
}

// HostFromURLOrHost parses a URL or bare hostname and returns lowercase hostname.
func HostFromURLOrHost(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty URL")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	h := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if h == "" {
		return "", fmt.Errorf("empty host")
	}
	return h, nil
}

// CollectToolURLHosts extracts hostnames from tool execution configs (rest, graphql, webhook, flow).
func CollectToolURLHosts(snapshot *models.Server) []string {
	if snapshot == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		h, err := HostFromURLOrHost(s)
		if err != nil || h == "" {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	for _, t := range snapshot.Tools {
		if len(t.ExecutionConfig) == 0 {
			continue
		}
		var cfg map[string]json.RawMessage
		if err := json.Unmarshal(t.ExecutionConfig, &cfg); err != nil {
			continue
		}
		if raw, ok := cfg["url"]; ok {
			var u string
			if json.Unmarshal(raw, &u) == nil && strings.TrimSpace(u) != "" {
				add(u)
			}
		}
		if raw, ok := cfg["auth"]; ok {
			var auth struct {
				OAuth2 *struct {
					TokenURL string `json:"tokenUrl"`
				} `json:"oauth2"`
			}
			if json.Unmarshal(raw, &auth) == nil && auth.OAuth2 != nil {
				add(auth.OAuth2.TokenURL)
			}
		}
		// Flow nodes: api steps with url
		if raw, ok := cfg["nodes"]; ok {
			var nodes []interface{}
			if json.Unmarshal(raw, &nodes) != nil {
				continue
			}
			for _, n := range nodes {
				m, ok := n.(map[string]interface{})
				if !ok {
					continue
				}
				if typ, _ := m["type"].(string); typ != "api" {
					continue
				}
				data, _ := m["data"].(map[string]interface{})
				if data == nil {
					continue
				}
				c, _ := data["config"].(map[string]interface{})
				if c == nil {
					continue
				}
				if u, ok := c["url"].(string); ok {
					add(u)
				}
			}
		}
	}
	return out
}

// HostsFromEnvProfilesJSON parses env_profiles for base_url keys and returns hostnames.
func HostsFromEnvProfilesJSON(envProfilesJSON []byte, profileKey string) []string {
	if len(envProfilesJSON) == 0 || strings.TrimSpace(profileKey) == "" {
		return nil
	}
	var profiles map[string]models.EnvProfile
	if err := json.Unmarshal(envProfilesJSON, &profiles); err != nil {
		return nil
	}
	want := strings.TrimSpace(profileKey)
	for k, v := range profiles {
		if !strings.EqualFold(strings.TrimSpace(k), want) {
			continue
		}
		if strings.TrimSpace(v.BaseURL) == "" {
			return nil
		}
		if h, err := HostFromURLOrHost(v.BaseURL); err == nil {
			return []string{h}
		}
		return nil
	}
	return nil
}
