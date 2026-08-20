package passkey

import (
	"fmt"
	"github.com/NookMux/NookMux/internal/store/passkey"
	"github.com/NookMux/NookMux/internal/store/user"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
	"strconv"
	"strings"
)

type WebAuthnUser struct {
	user       *userstore.User
	credential *passkeystore.PasskeyCredential
}

func NewWebAuthnUser(user *userstore.User, credential *passkeystore.PasskeyCredential) *WebAuthnUser {
	return &WebAuthnUser{user: user, credential: credential}
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	if u == nil || u.user == nil {
		return nil
	}
	return []byte(strconv.Itoa(u.user.Id))
}

func (u *WebAuthnUser) WebAuthnName() string {
	if u == nil || u.user == nil {
		return ""
	}
	name := strings.TrimSpace(u.user.Username)
	if name == "" {
		return fmt.Sprintf("user-%d", u.user.Id)
	}
	return name
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	if u == nil || u.user == nil {
		return ""
	}
	display := strings.TrimSpace(u.user.DisplayName)
	if display != "" {
		return display
	}
	return u.WebAuthnName()
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	if u == nil {
		return nil
	}
	if u.credential != nil {
		cred := u.credential.ToWebAuthnCredential()
		return []webauthn.Credential{cred}
	}
	credentials, err := passkeystore.GetPasskeysByUserID(u.user.Id)
	if err != nil || len(credentials) == 0 {
		return nil
	}
	result := make([]webauthn.Credential, 0, len(credentials))
	for _, c := range credentials {
		result = append(result, c.ToWebAuthnCredential())
	}
	return result
}

func (u *WebAuthnUser) ModelUser() *userstore.User {
	if u == nil {
		return nil
	}
	return u.user
}

func (u *WebAuthnUser) PasskeyCredential() *passkeystore.PasskeyCredential {
	if u == nil {
		return nil
	}
	return u.credential
}
