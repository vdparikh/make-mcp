// Package mcpvalidate enforces MCP specification conventions for tool names.
package mcpvalidate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ErrInvalidToolName indicates the name does not meet MCP naming rules.
var ErrInvalidToolName = errors.New("invalid tool name")

// ErrDuplicateToolName indicates another tool on the same server already uses this name (case-sensitive).
var ErrDuplicateToolName = errors.New("duplicate tool name")

// Allowed: A–Z, a–z, 0–9, underscore, hyphen, dot; length 1–128 (MCP guidance).
var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

// ValidateToolName checks MCP-recommended tool name rules (length, charset, no spaces).
func ValidateToolName(name string) error {
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 128 {
		return fmt.Errorf("%w: must be 1–128 characters", ErrInvalidToolName)
	}
	if !toolNamePattern.MatchString(name) {
		return fmt.Errorf("%w: use only letters, digits, underscore (_), hyphen (-), and dot (.) — no spaces or other characters", ErrInvalidToolName)
	}
	return nil
}

// SanitizeToolName maps disallowed characters to underscores, collapses repeats, trims edges, truncates to 128 runes.
// Used for OpenAPI-derived names so imports stay valid; prefer ValidateToolName for user-entered names.
func SanitizeToolName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	out = strings.Trim(out, "._-")
	if out == "" {
		return "tool"
	}
	if utf8.RuneCountInString(out) > 128 {
		out = truncateRunes(out, 128)
	}
	return out
}

// EnsureUniqueToolName returns name if unused; otherwise name_2, name_3, … (each ≤128 runes).
func EnsureUniqueToolName(base string, used map[string]struct{}) string {
	name := base
	n := 2
	for {
		if _, ok := used[name]; !ok {
			used[name] = struct{}{}
			return name
		}
		suffix := fmt.Sprintf("_%d", n)
		n++
		prefix := truncateRunes(base, 128-utf8.RuneCountInString(suffix))
		if prefix == "" {
			prefix = "tool"
		}
		name = prefix + suffix
		if utf8.RuneCountInString(name) > 128 {
			name = truncateRunes(name, 128)
		}
	}
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	i := 0
	for _, r := range s {
		if i >= max {
			break
		}
		b.WriteRune(r)
		i++
	}
	return b.String()
}
