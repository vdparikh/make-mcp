package webauthn

import (
	"encoding/json"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// WebAuthnUser adapts models.User to webauthn.User and holds credentials loaded from DB.
type WebAuthnUser struct {
	*models.User
	Credentials []webauthn.Credential
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	return []byte(u.User.ID)
}

func (u *WebAuthnUser) WebAuthnName() string {
	return u.User.Email
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.User.Name
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// NewWebAuthnUser creates a WebAuthnUser from a model user and credential JSON bytes (one per credential).
func NewWebAuthnUser(user *models.User, credentialDataList [][]byte) (*WebAuthnUser, error) {
	creds := make([]webauthn.Credential, 0, len(credentialDataList))
	for _, data := range credentialDataList {
		var c webauthn.Credential
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		creds = append(creds, c)
	}
	return &WebAuthnUser{User: user, Credentials: creds}, nil
}
