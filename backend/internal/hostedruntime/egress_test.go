package hostedruntime

import "testing"

func TestHostAllowed(t *testing.T) {
	tests := []struct {
		host  string
		rules []string
		want  bool
	}{
		{"api.example.com", []string{"api.example.com"}, true},
		{"API.EXAMPLE.COM", []string{"api.example.com"}, true},
		{"sub.api.example.com", []string{"*.example.com"}, true},
		{"example.com", []string{"*.example.com"}, false},
		{"evil.com", []string{"api.example.com"}, false},
		{"a.b.c", []string{"*.c"}, true},
	}
	for _, tt := range tests {
		if got := HostAllowed(tt.host, tt.rules); got != tt.want {
			t.Errorf("HostAllowed(%q, %v) = %v, want %v", tt.host, tt.rules, got, tt.want)
		}
	}
}

func TestHostFromURLOrHost(t *testing.T) {
	h, err := HostFromURLOrHost("https://api.foo.bar/v1")
	if err != nil || h != "api.foo.bar" {
		t.Fatalf("got %q %v", h, err)
	}
	h2, err := HostFromURLOrHost("registry.npmjs.org")
	if err != nil || h2 != "registry.npmjs.org" {
		t.Fatalf("got %q %v", h2, err)
	}
}

func TestResolveTierAndCaps(t *testing.T) {
	plat := DefaultPlatformLimits()
	r, err := Resolve(UserConfig{IsolationTier: "strict"}, plat)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tier != TierStrict {
		t.Fatalf("tier %s", r.Tier)
	}
	if r.MemoryBytes != plat.Tiers[TierStrict].MemoryMB*1024*1024 {
		t.Fatalf("memory %d", r.MemoryBytes)
	}
	_, err = Resolve(UserConfig{IsolationTier: "nope"}, plat)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveEgressDeny(t *testing.T) {
	plat := DefaultPlatformLimits()
	r, err := Resolve(UserConfig{
		EgressPolicy: EgressDenyDefault,
		Allowlist:    []string{"api.example.com", "*.foo.org"},
	}, plat)
	if err != nil {
		t.Fatal(err)
	}
	if r.EgressPolicy != EgressDenyDefault {
		t.Fatal(r.EgressPolicy)
	}
	if len(r.EgressAllowlist) != 2 {
		t.Fatal(len(r.EgressAllowlist))
	}
}
