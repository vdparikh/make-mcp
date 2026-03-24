package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOAuthParseTokenRequest_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	body := `{"grant_type":"authorization_code","code":"c1","redirect_uri":"http://127.0.0.1:6274/oauth/callback"}`
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/token?server_id=srv-a", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	gt, code, redir, rt, sid, err := oauthParseTokenRequest(c)
	if err != nil {
		t.Fatal(err)
	}
	if gt != "authorization_code" || code != "c1" || redir != "http://127.0.0.1:6274/oauth/callback" || rt != "" || sid != "srv-a" {
		t.Fatalf("parse: grant=%q code=%q redir=%q rt=%q sid=%q", gt, code, redir, rt, sid)
	}
}

func TestOAuthRedirectURIMatches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		stored   string
		provided string
		want     bool
	}{
		{"http://127.0.0.1:6274/oauth/callback", "http://127.0.0.1:6274/oauth/callback", true},
		{"http://127.0.0.1:6274/oauth/callback", "http://127.0.0.1:6274/oauth/callback/", true},
		{"http://127.0.0.1:6274/oauth/callback", "http://127.0.0.1:6275/oauth/callback", false},
		{"http://127.0.0.1:6274/cb?x=1", "http://127.0.0.1:6274/cb?x=1", true},
		{"http://127.0.0.1:6274/cb?x=1", "http://127.0.0.1:6274/cb?x=2", false},
	}
	for _, tc := range cases {
		got := oauthRedirectURIMatches(tc.stored, tc.provided)
		if got != tc.want {
			t.Errorf("oauthRedirectURIMatches(%q, %q) = %v, want %v", tc.stored, tc.provided, got, tc.want)
		}
	}
}

func TestOAuthBFFIssuerURL(t *testing.T) {
	t.Parallel()
	if got, want := oauthBFFIssuerURL("http://127.0.0.1:8080", "abc-uuid"), "http://127.0.0.1:8080/api/oauth/bff/abc-uuid"; got != want {
		t.Fatalf("oauthBFFIssuerURL = %q, want %q", got, want)
	}
	if got := oauthBFFIssuerURL("http://127.0.0.1:8080/", "x"); got != "http://127.0.0.1:8080/api/oauth/bff/x" {
		t.Fatalf("oauthBFFIssuerURL trailing base: %q", got)
	}
}

func TestNormalizeHostedResourceURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{
			"http://127.0.0.1:8080/api/api/users/7e119de7-91ac-4fa6-9adc-de7bd6e5afb8/slug",
			"http://127.0.0.1:8080/api/users/7e119de7-91ac-4fa6-9adc-de7bd6e5afb8/slug",
		},
		{
			"http://127.0.0.1:8080/api/users/u/s",
			"http://127.0.0.1:8080/api/users/u/s",
		},
	}
	for _, tc := range cases {
		got := normalizeHostedResourceURL(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeHostedResourceURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOAuthBFFStateRoundTrip(t *testing.T) {
	prev := os.Getenv("JWT_SECRET")
	_ = os.Setenv("JWT_SECRET", "test-secret-for-oauth-bff-state-only")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("JWT_SECRET")
		} else {
			_ = os.Setenv("JWT_SECRET", prev)
		}
	})

	s, err := signOAuthState("srv-1", "http://127.0.0.1:9/callback", "jam-state-xyz")
	if err != nil {
		t.Fatalf("signOAuthState: %v", err)
	}
	claims, err := parseOAuthState(s)
	if err != nil {
		t.Fatalf("parseOAuthState: %v", err)
	}
	if claims.ServerID != "srv-1" || claims.ReturnRedirectURI != "http://127.0.0.1:9/callback" || claims.ReturnState != "jam-state-xyz" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}
