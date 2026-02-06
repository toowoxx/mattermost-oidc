// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

package openid

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// OIDCClaims represents the standard OIDC UserInfo claims.
// See: https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
type OIDCClaims struct {
	// Sub is the unique identifier for the user (REQUIRED).
	// This MUST be used as AuthData since it's stable and unique per issuer.
	Sub string `json:"sub"`

	// Email is the user's email address.
	Email string `json:"email"`

	// EmailVerified indicates whether the email has been verified.
	EmailVerified bool `json:"email_verified"`

	// PreferredUsername is the user's preferred username.
	// Falls back to email local part if not provided.
	PreferredUsername string `json:"preferred_username"`

	// GivenName is the user's first name.
	GivenName string `json:"given_name"`

	// FamilyName is the user's last name.
	FamilyName string `json:"family_name"`

	// Name is the user's full display name.
	// Used as fallback when GivenName/FamilyName are not provided.
	Name string `json:"name"`

	// Picture is a URL to the user's profile picture.
	Picture string `json:"picture"`
}

// ParseOIDCClaims decodes OIDC claims from a JSON reader.
func ParseOIDCClaims(data io.Reader) (*OIDCClaims, error) {
	var claims OIDCClaims
	if err := json.NewDecoder(data).Decode(&claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// Validate checks that required claims are present and valid.
func (c *OIDCClaims) Validate() error {
	if c.Sub == "" {
		return errors.New("OIDC sub claim is required but missing")
	}
	if c.Email == "" {
		return errors.New("OIDC email claim is required but missing")
	}
	return nil
}

// ToUser converts OIDC claims to a Mattermost user model.
// The returned user is suitable for use with the OAuth flow.
func (c *OIDCClaims) ToUser(logger mlog.LoggerIFace) *model.User {
	user := &model.User{}

	// Username: prefer preferred_username, fallback to email local part
	username := c.PreferredUsername
	if username == "" {
		// Extract local part from email (before @)
		if atIdx := strings.Index(c.Email, "@"); atIdx > 0 {
			username = c.Email[:atIdx]
		} else {
			username = c.Email
		}
	}
	user.Username = model.CleanUsername(logger, username)

	// Name: prefer structured claims, fallback to splitting full name
	if c.GivenName != "" || c.FamilyName != "" {
		user.FirstName = c.GivenName
		user.LastName = c.FamilyName
	} else if c.Name != "" {
		// Split full name into first/last
		parts := strings.SplitN(c.Name, " ", 2)
		user.FirstName = parts[0]
		if len(parts) > 1 {
			user.LastName = parts[1]
		}
	}

	// Email: MUST be lowercase
	user.Email = strings.ToLower(c.Email)

	// Auth: use OIDC sub claim as stable identifier
	// NEVER use email or username as AuthData since they can change
	sub := c.Sub
	user.AuthData = &sub
	user.AuthService = model.ServiceOpenid

	// Pass through the IdP's email_verified claim
	user.EmailVerified = c.EmailVerified

	return user
}

// GetAuthData returns the OIDC sub claim, which should be used as AuthData.
func (c *OIDCClaims) GetAuthData() string {
	return c.Sub
}
