package hostedruntime

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Tier names map to CPU/memory/pids presets (operator-tunable caps in config).
const (
	TierStandard   = "standard"
	TierRestricted = "restricted"
	TierStrict     = "strict"
)

// UserConfig is stored in servers.hosted_runtime_config (JSON).
type UserConfig struct {
	IsolationTier string   `json:"isolation_tier,omitempty"`
	MemoryMB      int64    `json:"memory_mb,omitempty"`
	NanoCPUs      int64    `json:"nano_cpus,omitempty"`
	PidsLimit     int64    `json:"pids_limit,omitempty"`
	EgressPolicy  string   `json:"egress_policy,omitempty"` // allow_all | deny_default
	Allowlist     []string `json:"egress_allowlist,omitempty"`
}

// PlatformLimits caps user overrides and defines tier defaults (from YAML).
type PlatformLimits struct {
	MaxMemoryMB  int64
	MaxNanoCPUs  int64
	MaxPidsLimit int64
	Tiers        map[string]TierPreset
}

// TierPreset is one isolation tier.
type TierPreset struct {
	MemoryMB  int64
	NanoCPUs  int64
	PidsLimit int64
}

// Resolved is applied to Docker and container env (egress).
type Resolved struct {
	Tier              string
	MemoryBytes       int64
	NanoCPUs          int64
	PidsLimit         int64
	EgressPolicy      string
	EgressAllowlist   []string // normalized host rules for MCP_EGRESS_ALLOWLIST
}

// DefaultPlatformLimits returns conservative defaults when YAML omits hosted.runtime_isolation.
func DefaultPlatformLimits() PlatformLimits {
	return PlatformLimits{
		MaxMemoryMB:  4096,
		MaxNanoCPUs:  2_000_000_000,
		MaxPidsLimit: 512,
		Tiers: map[string]TierPreset{
			TierStandard: {
				MemoryMB:  512,
				NanoCPUs:  500_000_000,
				PidsLimit: 128,
			},
			TierRestricted: {
				MemoryMB:  384,
				NanoCPUs:  350_000_000,
				PidsLimit: 96,
			},
			TierStrict: {
				MemoryMB:  256,
				NanoCPUs:  250_000_000,
				PidsLimit: 64,
			},
		},
	}
}

// ParseUserConfig unmarshals JSON; empty input yields zero struct (defaults applied in Resolve).
func ParseUserConfig(raw json.RawMessage) (UserConfig, error) {
	if len(raw) == 0 {
		return UserConfig{}, nil
	}
	var c UserConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return UserConfig{}, fmt.Errorf("hosted_runtime_config: %w", err)
	}
	return c, nil
}

// Resolve merges user config, tier defaults, and platform caps.
func Resolve(uc UserConfig, plat PlatformLimits) (*Resolved, error) {
	if plat.Tiers == nil {
		plat = DefaultPlatformLimits()
	}
	tier := strings.TrimSpace(strings.ToLower(uc.IsolationTier))
	if tier == "" {
		tier = TierStandard
	}
	preset, ok := plat.Tiers[tier]
	if !ok {
		return nil, fmt.Errorf("unknown isolation_tier %q (use standard, restricted, strict)", tier)
	}

	memMB := preset.MemoryMB
	if uc.MemoryMB > 0 {
		memMB = uc.MemoryMB
	}
	nano := preset.NanoCPUs
	if uc.NanoCPUs > 0 {
		nano = uc.NanoCPUs
	}
	pids := preset.PidsLimit
	if uc.PidsLimit > 0 {
		pids = uc.PidsLimit
	}

	maxMB := plat.MaxMemoryMB
	if maxMB <= 0 {
		maxMB = DefaultPlatformLimits().MaxMemoryMB
	}
	maxNano := plat.MaxNanoCPUs
	if maxNano <= 0 {
		maxNano = DefaultPlatformLimits().MaxNanoCPUs
	}
	maxPids := plat.MaxPidsLimit
	if maxPids <= 0 {
		maxPids = DefaultPlatformLimits().MaxPidsLimit
	}

	if memMB > maxMB {
		memMB = maxMB
	}
	if nano > maxNano {
		nano = maxNano
	}
	if pids > maxPids {
		pids = maxPids
	}
	if memMB < 64 {
		memMB = 64
	}
	if nano < 50_000_000 {
		nano = 50_000_000
	}
	if pids < 32 {
		pids = 32
	}

	policy := strings.TrimSpace(strings.ToLower(uc.EgressPolicy))
	if policy == "" {
		policy = EgressAllowAll
	}
	switch policy {
	case EgressAllowAll, EgressDenyDefault:
	default:
		return nil, fmt.Errorf("egress_policy must be %q or %q", EgressAllowAll, EgressDenyDefault)
	}

	var allow []string
	for _, a := range uc.Allowlist {
		s := strings.TrimSpace(a)
		if s == "" {
			continue
		}
		allow = append(allow, s)
	}

	return &Resolved{
		Tier:            tier,
		MemoryBytes:     memMB * 1024 * 1024,
		NanoCPUs:        nano,
		PidsLimit:       pids,
		EgressPolicy:    policy,
		EgressAllowlist: allow,
	}, nil
}

// MergeEgressAllowlist deduplicates host rules for env (comma-separated).
func MergeEgressAllowlist(parts ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, p := range parts {
		for _, s := range p {
			t := strings.TrimSpace(s)
			if t == "" {
				continue
			}
			k := strings.ToLower(t)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}
