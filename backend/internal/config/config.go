// Package config loads application settings from YAML plus required environment secrets.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/llm"
	"gopkg.in/yaml.v3"
)

// HostedRuntimeIsolationYAML defines operator limits for hosted runtime isolation (optional; zeros use code defaults).
type HostedRuntimeIsolationYAML struct {
	MaxMemoryMB  int64                            `yaml:"max_memory_mb"`
	MaxNanoCPUs  int64                            `yaml:"max_nano_cpus"`
	MaxPidsLimit int64                            `yaml:"max_pids"`
	Tiers        map[string]HostedRuntimeTierYAML `yaml:"tiers"`
}

// HostedRuntimeTierYAML is one named tier (standard, restricted, strict).
type HostedRuntimeTierYAML struct {
	MemoryMB int64 `yaml:"memory_mb"`
	NanoCPUs int64 `yaml:"nano_cpus"`
	Pids     int64 `yaml:"pids"`
}

// Config is the full application configuration (non-secret values from YAML).
type Config struct {
	Server struct {
		ListenHost string `yaml:"listen_host"`
		ListenPort int    `yaml:"listen_port"`
		Debug      bool   `yaml:"debug"`
	} `yaml:"server"`

	CORS struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"cors"`

	WebAuthn struct {
		RPID      string   `yaml:"rp_id"`
		RPOrigins []string `yaml:"rp_origins"`
	} `yaml:"webauthn"`

	Hosted struct {
		// ContainerDialHost is the host the API uses to reach published MCP containers (Docker host port binding).
		ContainerDialHost string `yaml:"container_dial_host"`
		// ContainerBindHost is the Docker port binding HostIP (who may connect to the mapped port).
		ContainerBindHost string `yaml:"container_bind_host"`
		// GeneratedServerPublicHostIP is the fallback host segment in generated server.ts when no Host header is present.
		GeneratedServerPublicHostIP string `yaml:"generated_server_public_host_ip"`
		// ObservabilityDockerHostAlias is substituted into ingest URLs so containers can reach the API (e.g. host.docker.internal).
		ObservabilityDockerHostAlias string `yaml:"observability_docker_host_alias"`
		// ObservabilityReplaceAPIHosts lists API hostnames (from public URL) to rewrite to ObservabilityDockerHostAlias for container env.
		ObservabilityReplaceAPIHosts []string `yaml:"observability_replace_api_hosts"`
		// RuntimeIsolation caps per-tier resources and user overrides for hosted MCP containers.
		RuntimeIsolation HostedRuntimeIsolationYAML `yaml:"runtime_isolation"`
	} `yaml:"hosted"`

	URLs struct {
		// MCPHostedBaseURL if non-empty overrides request-derived base for hosted MCP URLs (trim trailing slash).
		MCPHostedBaseURL string `yaml:"mcp_hosted_base_url"`
		// MCPObservabilityIngestBaseURL if non-empty overrides observability ingest base for hosted runtimes.
		MCPObservabilityIngestBaseURL string `yaml:"mcp_observability_ingest_base_url"`
	} `yaml:"urls"`

	Paths struct {
		DocsDir string `yaml:"docs_dir"`
	} `yaml:"paths"`

	// LLM provider settings (Groq, OpenAI-compatible, etc.). API keys come from env (see api_key_env).
	LLM llm.Config `yaml:"llm"`
}

// Load reads YAML from MAKE_MCP_CONFIG or config/config.yaml / config.yaml.
func Load() (*Config, error) {
	path := strings.TrimSpace(os.Getenv("MAKE_MCP_CONFIG"))
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			wd = "."
		}
		candidates := []string{
			filepath.Join(wd, "config", "config.yaml"),
			filepath.Join(wd, "..", "config", "config.yaml"),
			filepath.Join(wd, "config.yaml"),
		}
		for _, c := range candidates {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				path = c
				break
			}
		}
	}
	if path == "" {
		return nil, fmt.Errorf("config file not found: set MAKE_MCP_CONFIG or create config/config.yaml (see config/config.example.yaml)")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	c.applyEnvOverrides()
	return &c, nil
}

func (c *Config) validate() error {
	if c.Server.ListenPort <= 0 || c.Server.ListenPort > 65535 {
		return fmt.Errorf("config: server.listen_port must be between 1 and 65535")
	}
	if strings.TrimSpace(c.Server.ListenHost) == "" {
		return fmt.Errorf("config: server.listen_host is required")
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("config: cors.allowed_origins must be non-empty")
	}
	if strings.TrimSpace(c.WebAuthn.RPID) == "" {
		return fmt.Errorf("config: webauthn.rp_id is required")
	}
	if len(c.WebAuthn.RPOrigins) == 0 {
		return fmt.Errorf("config: webauthn.rp_origins must be non-empty")
	}
	for i, o := range c.WebAuthn.RPOrigins {
		if strings.TrimSpace(o) == "" {
			return fmt.Errorf("config: webauthn.rp_origins[%d] is empty", i)
		}
	}
	if strings.TrimSpace(c.Hosted.ContainerDialHost) == "" {
		return fmt.Errorf("config: hosted.container_dial_host is required")
	}
	if strings.TrimSpace(c.Hosted.ContainerBindHost) == "" {
		return fmt.Errorf("config: hosted.container_bind_host is required")
	}
	if strings.TrimSpace(c.Hosted.GeneratedServerPublicHostIP) == "" {
		return fmt.Errorf("config: hosted.generated_server_public_host_ip is required")
	}
	if strings.TrimSpace(c.Hosted.ObservabilityDockerHostAlias) == "" {
		return fmt.Errorf("config: hosted.observability_docker_host_alias is required")
	}
	if len(c.Hosted.ObservabilityReplaceAPIHosts) == 0 {
		return fmt.Errorf("config: hosted.observability_replace_api_hosts must be non-empty")
	}
	return nil
}

func (c *Config) applyEnvOverrides() {
	if v := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ID")); v != "" {
		c.WebAuthn.RPID = v
	}
	if v := strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ORIGINS")); v != "" {
		parts := strings.Split(v, ",")
		var origins []string
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				origins = append(origins, t)
			}
		}
		if len(origins) > 0 {
			c.WebAuthn.RPOrigins = origins
		}
	}
	if v := strings.TrimSpace(os.Getenv("MCP_HOSTED_BASE_URL")); v != "" {
		c.URLs.MCPHostedBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MCP_OBSERVABILITY_INGEST_BASE_URL")); v != "" {
		c.URLs.MCPObservabilityIngestBaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("DOCS_DIR")); v != "" {
		c.Paths.DocsDir = v
	}
	if v := strings.TrimSpace(os.Getenv("DEBUG")); v != "" {
		c.Server.Debug = v == "true" || v == "1" || v == "yes"
	}
	if v := strings.TrimSpace(os.Getenv("PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p <= 65535 {
			c.Server.ListenPort = p
		}
	}
}

// ListenAddr returns host:port for gin / net.Listen.
func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", strings.TrimSpace(c.Server.ListenHost), c.Server.ListenPort)
}

// ShouldRewriteObservabilityHost returns true if the API hostname should be rewritten for container reachability.
func (c *Config) ShouldRewriteObservabilityHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return false
	}
	for _, x := range c.Hosted.ObservabilityReplaceAPIHosts {
		if strings.EqualFold(strings.TrimSpace(x), h) {
			return true
		}
	}
	return false
}
