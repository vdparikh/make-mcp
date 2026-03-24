package mcpvalidate

import (
	"strings"
	"testing"
)

func TestValidateToolName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"valid snake", "get_user", false},
		{"valid mixed case", "DATA_EXPORT_v2", false},
		{"valid dots", "admin.tools.list", false},
		{"valid hyphen", "get-user", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 129), true},
		{"space", "bad name", true},
		{"comma", "a,b", true},
		{"slash", "a/b", true},
		{"unicode", "工具", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateToolName(tc.in)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected: %v", err)
			}
		})
	}
}

func TestSanitizeToolName(t *testing.T) {
	t.Parallel()
	if got := SanitizeToolName("get/User"); got != "get_User" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeToolName(""); got != "tool" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureUniqueToolName(t *testing.T) {
	t.Parallel()
	used := make(map[string]struct{})
	a := EnsureUniqueToolName("x", used)
	b := EnsureUniqueToolName("x", used)
	if a != "x" || b != "x_2" {
		t.Fatalf("got %q %q", a, b)
	}
}
