package hostedsecurity

import (
	"net/http"
	"testing"
)

func TestIPAllowed(t *testing.T) {
	cases := []struct {
		ip   string
		list []string
		want bool
	}{
		{"127.0.0.1", nil, true},
		{"127.0.0.1", []string{}, true},
		{"127.0.0.1", []string{"127.0.0.1/32"}, true},
		{"10.0.0.5", []string{"10.0.0.0/8"}, true},
		{"192.168.1.1", []string{"10.0.0.0/8"}, false},
	}
	for _, tc := range cases {
		if got := IPAllowed(tc.ip, tc.list); got != tc.want {
			t.Fatalf("IPAllowed(%q, %v) = %v want %v", tc.ip, tc.list, got, tc.want)
		}
	}
}

func TestClientIP(t *testing.T) {
	r := &http.Request{Header: http.Header{}, RemoteAddr: "192.168.1.2:1234"}
	r.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if got := ClientIP(r); got != "203.0.113.1" {
		t.Fatalf("ClientIP xff: got %q", got)
	}
}
