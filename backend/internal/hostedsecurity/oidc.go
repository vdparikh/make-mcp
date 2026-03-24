package hostedsecurity

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const oidcHTTPTimeout = 15 * time.Second

// VerifyOIDCJWT validates a Bearer JWT against issuer/audience using JWKS (discovered or explicit).
func VerifyOIDCJWT(ctx context.Context, issuer, audience, jwksURL, rawToken string) (sub, email string, err error) {
	issuer = strings.TrimSuffix(strings.TrimSpace(issuer), "/")
	if issuer == "" {
		return "", "", fmt.Errorf("oidc issuer is required")
	}
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", "", fmt.Errorf("missing bearer token")
	}
	jwks := strings.TrimSpace(jwksURL)
	if jwks == "" {
		jwks, err = discoverJWKSURL(ctx, issuer)
		if err != nil {
			return "", "", err
		}
	}
	keys, err := getJWKSKeys(ctx, jwks)
	if err != nil {
		return "", "", err
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg(), jwt.SigningMethodES256.Alg()}),
		jwt.WithIssuer(issuer),
	}
	if strings.TrimSpace(audience) != "" {
		opts = append(opts, jwt.WithAudience(audience))
	}
	parser := jwt.NewParser(opts...)
	tok, err := parser.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("jwt missing kid header")
		}
		alg, _ := t.Header["alg"].(string)
		pub, ok := keys[kid]
		if !ok {
			return nil, fmt.Errorf("jwt kid not in JWKS")
		}
		switch strings.ToUpper(alg) {
		case jwt.SigningMethodRS256.Alg():
			rsaPub, ok := pub.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("jwt alg RS256 does not match JWKS key type for kid %q", kid)
			}
			return rsaPub, nil
		case jwt.SigningMethodES256.Alg():
			ecPub, ok := pub.(*ecdsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("jwt alg ES256 does not match JWKS key type for kid %q", kid)
			}
			return ecPub, nil
		default:
			return nil, fmt.Errorf("unsupported jwt alg %q (supported: RS256, ES256)", alg)
		}
	})
	if err != nil {
		return "", "", err
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid jwt claims")
	}
	sub, _ = claims["sub"].(string)
	email, _ = claims["email"].(string)
	if strings.TrimSpace(sub) == "" {
		return "", "", fmt.Errorf("jwt missing sub")
	}
	return sub, email, nil
}

// OIDCVerifyErrorHint returns a short troubleshooting line for API responses when VerifyOIDCJWT fails.
func OIDCVerifyErrorHint(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	switch {
	case strings.Contains(s, "jwks contained no usable"):
		return "Keycloak JWKS must expose RSA or EC P-256 keys; if your realm uses another curve, set oidc.jwks_url from the Make MCP host (same network as Docker)."
	case strings.Contains(s, "iss") && (strings.Contains(s, "mismatch") || strings.Contains(s, "invalid")):
		return "Set hosted_security_config.oidc.issuer to exactly match the JWT iss claim (localhost vs 127.0.0.1 vs Docker hostname)."
	case strings.Contains(s, "audience"):
		return "Leave oidc.audience empty unless you configured an audience mapper in Keycloak; access tokens often use aud=account."
	case strings.Contains(s, "kid not in JWKS"):
		return "Key may have rotated; wait a few minutes or check Keycloak realm keys. Ensure RS256/ES256 matches the token."
	default:
		return ""
	}
}

func discoverJWKSURL(ctx context.Context, issuer string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: oidcHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openid-configuration: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("openid-configuration: status %d", resp.StatusCode)
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("parse openid-configuration: %w", err)
	}
	if strings.TrimSpace(doc.JWKSURI) == "" {
		return "", fmt.Errorf("openid-configuration missing jwks_uri")
	}
	return doc.JWKSURI, nil
}

type jwksDoc struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
		Use string `json:"use"`
	} `json:"keys"`
}

type cachedJWKS struct {
	keys map[string]any // *rsa.PublicKey or *ecdsa.PublicKey
	at   time.Time
}

const jwksTTL = 5 * time.Minute

var jwksCache sync.Map // jwksURL -> cachedJWKS

func getJWKSKeys(ctx context.Context, jwksURL string) (map[string]any, error) {
	jwksURL = strings.TrimSpace(jwksURL)
	if jwksURL == "" {
		return nil, fmt.Errorf("jwks url empty")
	}
	if v, ok := jwksCache.Load(jwksURL); ok {
		c := v.(cachedJWKS)
		if time.Since(c.at) < jwksTTL && len(c.keys) > 0 {
			return c.keys, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: oidcHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jwks: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("jwks: status %d", resp.StatusCode)
	}
	var doc jwksDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}
	out := make(map[string]any)
	for _, k := range doc.Keys {
		switch strings.ToUpper(k.Kty) {
		case "RSA":
			if k.Kid == "" || k.N == "" || k.E == "" {
				continue
			}
			pub, err := rsaPubFromJWK(k.N, k.E)
			if err != nil {
				continue
			}
			out[k.Kid] = pub
		case "EC":
			if k.Kid == "" || k.Crv != "P-256" || k.X == "" || k.Y == "" {
				continue
			}
			pub, err := ecPubP256FromJWK(k.X, k.Y)
			if err != nil {
				continue
			}
			out[k.Kid] = pub
		default:
			continue
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("jwks contained no usable signing keys (need RSA or EC P-256)")
	}
	jwksCache.Store(jwksURL, cachedJWKS{keys: out, at: time.Now()})
	return out, nil
}

func ecPubP256FromJWK(xB64, yB64 string) (*ecdsa.PublicKey, error) {
	xb, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, err
	}
	yb, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, err
	}
	curve := elliptic.P256()
	const p256CoordBytes = 32
	if len(xb) != p256CoordBytes || len(yb) != p256CoordBytes {
		return nil, fmt.Errorf("invalid P-256 coordinate length")
	}
	x := new(big.Int).SetBytes(xb)
	y := new(big.Int).SetBytes(yb)
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("ec point not on P-256")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func rsaPubFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nb)
	e := 0
	for _, b := range eb {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, fmt.Errorf("invalid exponent")
	}
	return &rsa.PublicKey{N: n, E: e}, nil
}
