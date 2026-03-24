package hostedsecurity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func pad32(b []byte) []byte {
	const n = 32
	if len(b) > n {
		return b[len(b)-n:]
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}

func TestECPubP256FromJWK_roundtrip(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	xStr := base64.RawURLEncoding.EncodeToString(pad32(key.PublicKey.X.Bytes()))
	yStr := base64.RawURLEncoding.EncodeToString(pad32(key.PublicKey.Y.Bytes()))
	pub, err := ecPubP256FromJWK(xStr, yStr)
	if err != nil {
		t.Fatal(err)
	}
	if pub.X.Cmp(key.PublicKey.X) != 0 || pub.Y.Cmp(key.PublicKey.Y) != 0 {
		t.Fatal("roundtrip mismatch")
	}
}

func TestECPubP256FromJWK_errors(t *testing.T) {
	tests := []struct {
		name    string
		x       string
		y       string
		wantSub string
	}{
		{name: "bad base64", x: "!!!", y: "AAAA", wantSub: "illegal"},
		{name: "wrong length", x: "AA", y: "AA", wantSub: "length"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ecPubP256FromJWK(tc.x, tc.y)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantSub) {
				t.Fatalf("err %v want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestOIDCVerifyErrorHint(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{nil, ""},
		{strErr("jwks contained no usable signing keys"), "Keycloak JWKS must expose"},
		{strErr("claims Verification failed: claim iss mismatched"), "oidc.issuer"},
		{strErr("audience mismatch"), "audience empty"},
		{strErr("jwt kid not in JWKS"), "Key may have rotated"},
	}
	for _, tc := range tests {
		got := OIDCVerifyErrorHint(tc.err)
		if tc.want == "" {
			if got != "" {
				t.Errorf("OIDCVerifyErrorHint(nil-ish) = %q want empty", got)
			}
			continue
		}
		if !strings.Contains(got, tc.want) {
			t.Errorf("OIDCVerifyErrorHint(%v) = %q want substring %q", tc.err, got, tc.want)
		}
	}
}

type strErr string

func (e strErr) Error() string { return string(e) }
