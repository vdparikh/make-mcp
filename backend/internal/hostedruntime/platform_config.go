package hostedruntime

import (
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/config"
)

// PlatformLimitsFromYAML merges YAML-hosted.runtime_isolation onto defaults.
func PlatformLimitsFromYAML(y config.HostedRuntimeIsolationYAML) PlatformLimits {
	p := DefaultPlatformLimits()
	if y.MaxMemoryMB > 0 {
		p.MaxMemoryMB = y.MaxMemoryMB
	}
	if y.MaxNanoCPUs > 0 {
		p.MaxNanoCPUs = y.MaxNanoCPUs
	}
	if y.MaxPidsLimit > 0 {
		p.MaxPidsLimit = y.MaxPidsLimit
	}
	if len(y.Tiers) > 0 {
		def := DefaultPlatformLimits().Tiers
		for name, t := range y.Tiers {
			n := strings.ToLower(strings.TrimSpace(name))
			if n == "" {
				continue
			}
			preset := TierPreset{
				MemoryMB:  t.MemoryMB,
				NanoCPUs:  t.NanoCPUs,
				PidsLimit: t.Pids,
			}
			if base, ok := def[n]; ok {
				if preset.MemoryMB <= 0 {
					preset.MemoryMB = base.MemoryMB
				}
				if preset.NanoCPUs <= 0 {
					preset.NanoCPUs = base.NanoCPUs
				}
				if preset.PidsLimit <= 0 {
					preset.PidsLimit = base.PidsLimit
				}
			}
			p.Tiers[n] = preset
		}
	}
	return p
}
