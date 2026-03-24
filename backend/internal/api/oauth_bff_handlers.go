package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vdparikh/make-mcp/backend/internal/database"
	"github.com/vdparikh/make-mcp/backend/internal/hostedsecurity"
)

var oauthHTTPClient = &http.Client{Timeout: 30 * time.Second}

// upstreamOIDCDocument is a subset of GET {issuer}/.well-known/openid-configuration used for MCP client validation.
type upstreamOIDCDocument struct {
	JWKSURI                          string   `json:"jwks_uri"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

type cachedUpstreamOIDC struct {
	doc *upstreamOIDCDocument
	at  time.Time
}

const upstreamOIDCCacheTTL = 30 * time.Minute

var upstreamOIDCCache sync.Map // issuer string -> cachedUpstreamOIDC

// discoverUpstreamOIDCDocument fetches the upstream IdP discovery document (cached). Used to populate jwks_uri and OIDC-required metadata fields for MCP clients (e.g. MCP Jam).
func discoverUpstreamOIDCDocument(ctx context.Context, issuer string) (*upstreamOIDCDocument, error) {
	issuer = strings.TrimSuffix(strings.TrimSpace(issuer), "/")
	if issuer == "" {
		return nil, fmt.Errorf("empty upstream issuer")
	}
	if v, ok := upstreamOIDCCache.Load(issuer); ok {
		c := v.(cachedUpstreamOIDC)
		if time.Since(c.at) < upstreamOIDCCacheTTL && c.doc != nil {
			return c.doc, nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return nil, err
	}
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("openid-configuration: status %d", resp.StatusCode)
	}
	var doc upstreamOIDCDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse openid-configuration: %w", err)
	}
	if strings.TrimSpace(doc.JWKSURI) == "" {
		doc.JWKSURI = issuer + "/protocol/openid-connect/certs"
	}
	if len(doc.SubjectTypesSupported) == 0 {
		doc.SubjectTypesSupported = []string{"public"}
	}
	if len(doc.IDTokenSigningAlgValuesSupported) == 0 {
		doc.IDTokenSigningAlgValuesSupported = []string{"RS256", "ES256"}
	}
	upstreamOIDCCache.Store(issuer, cachedUpstreamOIDC{doc: &doc, at: time.Now()})
	return &doc, nil
}

const oauthBriefCodeTTL = 5 * time.Minute

type oauthBriefEntry struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	RedirectURI  string
	ServerID     string
	CreatedAt    time.Time
}

var (
	oauthBriefMu    sync.Mutex
	oauthBriefCodes = make(map[string]*oauthBriefEntry)
)

func newOAuthBriefCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func oauthBriefPut(code string, e *oauthBriefEntry) {
	oauthBriefMu.Lock()
	oauthBriefCodes[code] = e
	oauthBriefMu.Unlock()
	time.AfterFunc(oauthBriefCodeTTL, func() {
		oauthBriefMu.Lock()
		delete(oauthBriefCodes, code)
		oauthBriefMu.Unlock()
	})
}

func oauthBriefTake(code string) *oauthBriefEntry {
	oauthBriefMu.Lock()
	defer oauthBriefMu.Unlock()
	e := oauthBriefCodes[code]
	delete(oauthBriefCodes, code)
	if e == nil {
		return nil
	}
	if time.Since(e.CreatedAt) > oauthBriefCodeTTL {
		return nil
	}
	return e
}

// oauthRedirectURIMatches compares redirect URIs from the authorize request and token exchange (RFC 6749).
func oauthRedirectURIMatches(stored, provided string) bool {
	stored = strings.TrimSpace(stored)
	provided = strings.TrimSpace(provided)
	if stored == provided {
		return true
	}
	su, e1 := url.Parse(stored)
	pu, e2 := url.Parse(provided)
	if e1 != nil || e2 != nil {
		return false
	}
	if !strings.EqualFold(su.Scheme, pu.Scheme) || su.Host != pu.Host {
		return false
	}
	p1 := strings.TrimSuffix(su.Path, "/")
	p2 := strings.TrimSuffix(pu.Path, "/")
	if p1 == "" {
		p1 = "/"
	}
	if p2 == "" {
		p2 = "/"
	}
	return p1 == p2 && su.RawQuery == pu.RawQuery
}

// normalizeHostedResourceURL collapses accidental /api/api/ (e.g. MCP clients that append "api" to a path that already includes /api/).
func normalizeHostedResourceURL(s string) string {
	s = strings.TrimSpace(s)
	for strings.Contains(s, "/api/api/") {
		s = strings.ReplaceAll(s, "/api/api/", "/api/")
	}
	return s
}

// resolveServerIDFromHostedResourceURL maps a hosted MCP URL (.../api/users/<owner>/<slug>) to a server id.
func (h *Handler) resolveServerIDFromHostedResourceURL(ctx context.Context, resource string) (string, error) {
	resource = normalizeHostedResourceURL(strings.TrimSpace(resource))
	if resource == "" {
		return "", fmt.Errorf("empty resource")
	}
	u, err := url.Parse(resource)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(parts)-2; i++ {
		if parts[i] == "users" && i+2 < len(parts) {
			uid, slug := parts[i+1], parts[i+2]
			s, err := h.db.GetServerByOwnerAndSlug(ctx, uid, slug)
			if err != nil || s == nil {
				continue
			}
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("could not resolve server from resource URL")
}

const oauthStateTTL = 15 * time.Minute

// oauthBFFStateClaims is signed into the state param for Keycloak round-trip.
type oauthBFFStateClaims struct {
	ServerID          string `json:"sid"`
	ReturnRedirectURI string `json:"rru"`
	ReturnState       string `json:"rs"`
	jwt.RegisteredClaims
}

func (h *Handler) oauthPublicBase(c *gin.Context) string {
	scheme := "https"
	if c.Request.TLS == nil && (c.Request.Header.Get("X-Forwarded-Proto") == "" || c.Request.Header.Get("X-Forwarded-Proto") == "http") {
		scheme = "http"
	}
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.Request.Host
	if fh := c.Request.Header.Get("X-Forwarded-Host"); fh != "" {
		host = fh
	}
	return scheme + "://" + host
}

// oauthCallbackURL is the redirect_uri Keycloak must allow. It must match hostedBaseURL (config mcp_hosted_base_url or request host) + /oauth/callback.
func (h *Handler) oauthCallbackURL(c *gin.Context) string {
	return h.hostedBaseURL(c) + "/oauth/callback"
}

func jwtSecretBytes() []byte {
	s := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if s == "" {
		return nil
	}
	return []byte(s)
}

func signOAuthState(serverID, returnRedirectURI, returnState string) (string, error) {
	key := jwtSecretBytes()
	if len(key) == 0 {
		return "", fmt.Errorf("JWT_SECRET is not set")
	}
	now := time.Now()
	claims := oauthBFFStateClaims{
		ServerID:          serverID,
		ReturnRedirectURI: returnRedirectURI,
		ReturnState:       returnState,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(oauthStateTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(key)
}

func parseOAuthState(state string) (*oauthBFFStateClaims, error) {
	key := jwtSecretBytes()
	if len(key) == 0 {
		return nil, fmt.Errorf("JWT_SECRET is not set")
	}
	tok, err := jwt.ParseWithClaims(state, &oauthBFFStateClaims{}, func(t *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*oauthBFFStateClaims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("invalid state")
	}
	return claims, nil
}

// oauthBFFIssuerURL is the OAuth authorization server issuer for one hosted server. Clients discover metadata at
// {issuer}/.well-known/openid-configuration (RFC 8414) — the path must include server_id because multiple servers share one API host.
func oauthBFFIssuerURL(base, serverID string) string {
	base = strings.TrimSuffix(strings.TrimSpace(base), "/")
	return base + "/api/oauth/bff/" + strings.TrimSpace(serverID)
}

// oauthServerIDFromRequest resolves server_id from path param, query, or hosted MCP resource URL (same rules as PRM).
func (h *Handler) oauthServerIDFromRequest(c *gin.Context) string {
	if id := strings.TrimSpace(c.Param("server_id")); id != "" {
		return id
	}
	if id := strings.TrimSpace(c.Query("server_id")); id != "" {
		return id
	}
	res := normalizeHostedResourceURL(strings.TrimSpace(c.Query("resource")))
	if res == "" {
		return ""
	}
	id, err := h.resolveServerIDFromHostedResourceURL(c.Request.Context(), res)
	if err != nil || id == "" {
		return ""
	}
	return id
}

// OAuthAuthorizationServerMetadata serves RFC 8414-style metadata at the API host so MCP clients discover /api/oauth/* (BFF to upstream IdP).
func (h *Handler) OAuthAuthorizationServerMetadata(c *gin.Context) {
	serverID := h.oauthServerIDFromRequest(c)
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing authorization server context: use the issuer URL from oauth-protected-resource (authorization_servers → .../api/oauth/bff/<server_id>) or pass ?server_id= or ?resource=<hosted MCP URL>",
		})
		return
	}
	srv, err := h.db.GetServer(c.Request.Context(), serverID)
	if err != nil || srv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	prof, err := hostedsecurity.Resolve(srv, srv.HostedSecurityConfig, c.GetHeader(hostedsecurity.HeaderEnv))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid hosted security configuration"})
		return
	}
	if prof.OAuthBFF == nil || !prof.OAuthBFF.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "oauth_bff is not enabled for this server"})
		return
	}
	base := strings.TrimSuffix(h.oauthPublicBase(c), "/")
	up := strings.TrimSuffix(strings.TrimSpace(prof.OAuthBFF.UpstreamIssuer), "/")
	authz := base + "/authorize?server_id=" + url.QueryEscape(serverID)
	token := base + "/api/oauth/token?server_id=" + url.QueryEscape(serverID)
	issuer := oauthBFFIssuerURL(base, serverID)

	oidcDoc, oerr := discoverUpstreamOIDCDocument(c.Request.Context(), up)
	jwksURI := up + "/protocol/openid-connect/certs"
	subjectTypes := []string{"public"}
	idTokenAlgs := []string{"RS256", "ES256"}
	if oerr == nil && oidcDoc != nil {
		if strings.TrimSpace(oidcDoc.JWKSURI) != "" {
			jwksURI = strings.TrimSpace(oidcDoc.JWKSURI)
		}
		if len(oidcDoc.SubjectTypesSupported) > 0 {
			subjectTypes = oidcDoc.SubjectTypesSupported
		}
		if len(oidcDoc.IDTokenSigningAlgValuesSupported) > 0 {
			idTokenAlgs = oidcDoc.IDTokenSigningAlgValuesSupported
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"issuer":                                issuer,
		"authorization_endpoint":                authz,
		"token_endpoint":                        token,
		"jwks_uri":                              jwksURI,
		"subject_types_supported":               subjectTypes,
		"id_token_signing_alg_values_supported": idTokenAlgs,
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic", "none"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
	})
}

// OAuthProtectedResourceMetadata serves RFC 9728-style PRM so clients find authorization_servers (this host).
func (h *Handler) OAuthProtectedResourceMetadata(c *gin.Context) {
	resource := normalizeHostedResourceURL(c.Query("resource"))
	serverID := strings.TrimSpace(c.Query("server_id"))
	if serverID == "" && resource != "" {
		if id, err := h.resolveServerIDFromHostedResourceURL(c.Request.Context(), resource); err == nil && id != "" {
			serverID = id
		}
	}
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id or resource (hosted MCP URL) query parameter is required"})
		return
	}
	srv, err := h.db.GetServer(c.Request.Context(), serverID)
	if err != nil || srv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	prof, err := hostedsecurity.Resolve(srv, srv.HostedSecurityConfig, c.GetHeader(hostedsecurity.HeaderEnv))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid hosted security configuration"})
		return
	}
	if prof.OAuthBFF == nil || !prof.OAuthBFF.Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "oauth_bff is not enabled for this server"})
		return
	}
	base := strings.TrimSuffix(h.oauthPublicBase(c), "/")
	if resource == "" {
		slug := database.ServerSlug(srv.Name)
		resource = base + "/api/users/" + srv.OwnerID + "/" + slug
	}
	resource = normalizeHostedResourceURL(resource)
	authServers := []string{oauthBFFIssuerURL(base, serverID)}
	c.JSON(http.StatusOK, gin.H{
		"resource":                 resource,
		"authorization_servers":    authServers,
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         []string{"openid", "profile", "email", "mcp:tools", "mcp:resources"},
	})
}

// OAuthAuthorize starts the BFF flow: redirects the browser to the upstream IdP (e.g. Keycloak).
func (h *Handler) OAuthAuthorize(c *gin.Context) {
	serverID := h.oauthServerIDFromRequest(c)
	if serverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server_id or resource query parameter is required"})
		return
	}
	srv, err := h.db.GetServer(c.Request.Context(), serverID)
	if err != nil || srv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	prof, err := hostedsecurity.Resolve(srv, srv.HostedSecurityConfig, c.GetHeader(hostedsecurity.HeaderEnv))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid hosted security configuration"})
		return
	}
	if prof.OAuthBFF == nil || !prof.OAuthBFF.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth_bff is not enabled for this server"})
		return
	}
	up := strings.TrimSuffix(strings.TrimSpace(prof.OAuthBFF.UpstreamIssuer), "/")
	if up == "" || strings.TrimSpace(prof.OAuthBFF.ClientID) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oauth_bff upstream_issuer and client_id are required"})
		return
	}

	returnRedirect := strings.TrimSpace(c.Query("redirect_uri"))
	if returnRedirect == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "redirect_uri is required (MCP client callback)"})
		return
	}
	retState := c.Query("state")
	scope := strings.TrimSpace(c.Query("scope"))
	if scope == "" {
		scope = "openid profile email"
	}

	stateJWT, err := signOAuthState(serverID, returnRedirect, retState)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cb := h.oauthCallbackURL(c)
	authPath := up + "/protocol/openid-connect/auth"
	q := url.Values{}
	q.Set("client_id", prof.OAuthBFF.ClientID)
	q.Set("redirect_uri", cb)
	q.Set("response_type", "code")
	q.Set("scope", scope)
	q.Set("state", stateJWT)
	loc := authPath + "?" + q.Encode()
	c.Redirect(http.StatusFound, loc)
}

// OAuthCallback exchanges the IdP authorization code, then redirects to the MCP client with a one-time code (query) for POST /api/oauth/token.
func (h *Handler) OAuthCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	errParam := strings.TrimSpace(c.Query("error"))
	if errParam != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errParam, "error_description": c.Query("error_description")})
		return
	}
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and state are required"})
		return
	}
	claims, err := parseOAuthState(state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired state"})
		return
	}
	srv, err := h.db.GetServer(c.Request.Context(), claims.ServerID)
	if err != nil || srv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	prof, err := hostedsecurity.Resolve(srv, srv.HostedSecurityConfig, c.GetHeader(hostedsecurity.HeaderEnv))
	if err != nil || prof.OAuthBFF == nil || !prof.OAuthBFF.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "oauth_bff is not enabled"})
		return
	}
	secret := oauthBFFClientSecret(prof.OAuthBFF)
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oauth_bff client secret is not configured (set client_secret_env)"})
		return
	}
	up := strings.TrimSuffix(strings.TrimSpace(prof.OAuthBFF.UpstreamIssuer), "/")
	tokenURL := up + "/protocol/openid-connect/token"
	cb := h.oauthCallbackURL(c)

	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("redirect_uri", cb)
	body.Set("client_id", prof.OAuthBFF.ClientID)
	body.Set("client_secret", secret)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build token request"})
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "token endpoint request failed", "details": err.Error()})
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream token exchange failed", "status": resp.StatusCode, "body": string(raw)})
		return
	}
	var tokResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(raw, &tokResp); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid token response"})
		return
	}
	if strings.TrimSpace(tokResp.AccessToken) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream returned no access_token"})
		return
	}

	brief, err := newOAuthBriefCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue exchange code"})
		return
	}
	oauthBriefPut(brief, &oauthBriefEntry{
		AccessToken:  tokResp.AccessToken,
		RefreshToken: tokResp.RefreshToken,
		ExpiresIn:    tokResp.ExpiresIn,
		RedirectURI:  claims.ReturnRedirectURI,
		ServerID:     claims.ServerID,
		CreatedAt:    time.Now(),
	})

	u, err := url.Parse(claims.ReturnRedirectURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid return redirect_uri in state"})
		return
	}
	q := u.Query()
	q.Set("code", brief)
	if claims.ReturnState != "" {
		q.Set("state", claims.ReturnState)
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	c.Redirect(http.StatusFound, u.String())
}

func oauthBFFClientSecret(bff *hostedsecurity.OAuthBFFConfig) string {
	if bff == nil {
		return ""
	}
	env := strings.TrimSpace(bff.ClientSecretEnv)
	if env != "" {
		return strings.TrimSpace(os.Getenv(env))
	}
	return ""
}

// oauthParseTokenRequest reads OAuth token endpoint parameters from JSON (MCP Jam) or form-urlencoded.
func oauthParseTokenRequest(c *gin.Context) (grantType, code, redirectURI, refreshToken, serverID string, err error) {
	serverID = strings.TrimSpace(c.Query("server_id"))
	ct := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
	if strings.Contains(ct, "application/json") {
		var body struct {
			GrantType    string `json:"grant_type"`
			Code         string `json:"code"`
			RedirectURI  string `json:"redirect_uri"`
			RefreshToken string `json:"refresh_token"`
			ServerID     string `json:"server_id"`
		}
		dec := json.NewDecoder(io.LimitReader(c.Request.Body, 1<<20))
		if jerr := dec.Decode(&body); jerr != nil && jerr != io.EOF {
			err = jerr
			return
		}
		if strings.TrimSpace(body.ServerID) != "" {
			serverID = strings.TrimSpace(body.ServerID)
		}
		return body.GrantType, body.Code, body.RedirectURI, body.RefreshToken, serverID, nil
	}
	if err = c.Request.ParseForm(); err != nil {
		return
	}
	if serverID == "" {
		serverID = strings.TrimSpace(c.PostForm("server_id"))
	}
	return c.PostForm("grant_type"), c.PostForm("code"), c.PostForm("redirect_uri"), c.PostForm("refresh_token"), serverID, nil
}

// OAuthToken implements the token endpoint: authorization_code (BFF one-time code) and refresh_token (proxied to IdP).
func (h *Handler) OAuthToken(c *gin.Context) {
	grantType, code, redirectURI, refreshToken, serverID, err := oauthParseTokenRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
		return
	}
	gt := strings.ToLower(strings.TrimSpace(grantType))
	switch gt {
	case "authorization_code":
		code = strings.TrimSpace(code)
		redir := strings.TrimSpace(redirectURI)
		if code == "" || redir == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "code and redirect_uri are required"})
			return
		}
		if serverID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "server_id is required on the token_endpoint URL"})
			return
		}
		ent := oauthBriefTake(code)
		if ent == nil || ent.ServerID != serverID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "invalid or expired authorization code"})
			return
		}
		if !oauthRedirectURIMatches(ent.RedirectURI, redir) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "redirect_uri does not match the authorize request"})
			return
		}
		body := gin.H{
			"access_token": ent.AccessToken,
			"token_type":   "Bearer",
		}
		if ent.ExpiresIn > 0 {
			body["expires_in"] = ent.ExpiresIn
		}
		if strings.TrimSpace(ent.RefreshToken) != "" {
			body["refresh_token"] = ent.RefreshToken
		}
		c.JSON(http.StatusOK, body)
	case "refresh_token":
		srv, err := h.db.GetServer(c.Request.Context(), serverID)
		if err != nil || srv == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		prof, err := hostedsecurity.Resolve(srv, srv.HostedSecurityConfig, c.GetHeader(hostedsecurity.HeaderEnv))
		if err != nil || prof.OAuthBFF == nil || !prof.OAuthBFF.Enabled {
			c.JSON(http.StatusBadRequest, gin.H{"error": "oauth_bff is not enabled"})
			return
		}
		secret := oauthBFFClientSecret(prof.OAuthBFF)
		if secret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "oauth_bff client secret is not configured"})
			return
		}
		rt := strings.TrimSpace(refreshToken)
		if rt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
			return
		}
		up := strings.TrimSuffix(strings.TrimSpace(prof.OAuthBFF.UpstreamIssuer), "/")
		tokenURL := up + "/protocol/openid-connect/token"
		body := url.Values{}
		body.Set("grant_type", "refresh_token")
		body.Set("refresh_token", rt)
		body.Set("client_id", prof.OAuthBFF.ClientID)
		body.Set("client_secret", secret)
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build refresh request"})
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := oauthHTTPClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		c.Data(resp.StatusCode, "application/json", raw)
	default:
		if gt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type", "error_description": "missing grant_type; send authorization_code or refresh_token (JSON or form body)"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type", "error_description": "use authorization_code or refresh_token"})
	}
}

// OAuthRegister is a placeholder for dynamic client registration (MCP clients may probe). Returns 501 so clients can fall back to pre-registered clients.
func (h *Handler) OAuthRegister(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":             "dynamic_client_registration_not_supported",
		"error_description": "Register OAuth clients at your IdP (e.g. Keycloak) and use oauth_bff client_id; optional server_id applies to hosted metadata.",
	})
}

// OAuthProtectedResourceMetadataByPath handles GET /.well-known/oauth-protected-resource/api/users/... (MCP clients append resource path).
func (h *Handler) OAuthProtectedResourceMetadataByPath(c *gin.Context) {
	p := strings.TrimPrefix(c.Param("resourcePath"), "/")
	base := strings.TrimSuffix(h.oauthPublicBase(c), "/")
	// Path suffix is usually "api/users/..." — do not prepend "/api/" again (would yield /api/api/users/...).
	// Some clients incorrectly send "api/api/users/..."; normalize after building.
	var resource string
	if strings.HasPrefix(p, "api/") {
		resource = base + "/" + p
	} else {
		resource = base + "/api/" + p
	}
	resource = normalizeHostedResourceURL(resource)
	q := c.Request.URL.Query()
	q.Set("resource", resource)
	c.Request.URL.RawQuery = q.Encode()
	h.OAuthProtectedResourceMetadata(c)
}

// oauthResourceMetadataURL returns the PRM URL for WWW-Authenticate (hosted MCP 401).
func (h *Handler) oauthResourceMetadataURL(c *gin.Context, serverID string) string {
	base := strings.TrimSuffix(h.oauthPublicBase(c), "/")
	return base + "/.well-known/oauth-protected-resource?server_id=" + url.QueryEscape(serverID)
}
