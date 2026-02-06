// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

// Package openid provides a generic OIDC SSO provider for Mattermost.
// It implements the einterfaces.OAuthProvider interface and can be used
// with any OIDC-compliant identity provider (Keycloak, Auth0, GitLab, Azure AD, Google).
//
// To use this provider, add a blank import in your main.go:
//
//	import _ "github.com/toowoxx/mattermost-oidc/openid"
//
// Then configure the OpenIdSettings in your Mattermost config.
package openid

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/public/shared/request"
	"github.com/mattermost/mattermost/server/v8/einterfaces"
)

// OpenIDProvider implements the einterfaces.OAuthProvider interface
// for generic OIDC authentication.
type OpenIDProvider struct{}

func init() {
	provider := &OpenIDProvider{}
	einterfaces.RegisterOAuthProvider(model.ServiceOpenid, provider)
}

// GetUserFromJSON parses the UserInfo response from the OIDC provider.
// This is called after the OAuth token exchange when fetching user info.
func (p *OpenIDProvider) GetUserFromJSON(rctx request.CTX, data io.Reader, tokenUser *model.User) (*model.User, error) {
	claims, err := ParseOIDCClaims(data)
	if err != nil {
		return nil, err
	}

	if err = claims.Validate(); err != nil {
		return nil, err
	}

	var logger mlog.LoggerIFace
	if rctx != nil {
		logger = rctx.Logger()
	}
	return claims.ToUser(logger), nil
}

// GetSSOSettings returns the OpenID settings from the Mattermost config.
func (p *OpenIDProvider) GetSSOSettings(_ request.CTX, config *model.Config, service string) (*model.SSOSettings, error) {
	return &config.OpenIdSettings, nil
}

// GetUserFromIdToken parses the JWT ID token to extract user claims.
// This is an optimization that allows extracting user info directly from the ID token
// instead of making an additional UserInfo endpoint request.
//
// Returns (nil, nil) if the ID token cannot be parsed or is empty,
// in which case the core will fall back to the UserInfo endpoint.
func (p *OpenIDProvider) GetUserFromIdToken(rctx request.CTX, idToken string) (*model.User, error) {
	if idToken == "" {
		return nil, nil
	}

	// Parse the JWT without signature verification.
	// The ID token was already validated by the OAuth flow (received from token endpoint over HTTPS).
	// We're only extracting claims here, not authenticating.
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		// Can't parse token - fall back to UserInfo endpoint
		return nil, nil
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil
	}

	// Convert map claims to our struct
	claims := mapClaimsToOIDCClaims(mapClaims)

	// Validate required claims
	if err = claims.Validate(); err != nil {
		// Missing required claims in ID token - fall back to UserInfo
		return nil, nil
	}

	return claims.ToUser(rctx.Logger()), nil
}

// IsSameUser compares two users to determine if they represent the same OIDC user.
//
// This is called by Mattermost's CreateOAuthUser after it finds an existing user
// by email. If IsSameUser returns true, the core calls UpdateAuthData to migrate
// the user to the new provider.
//
// Two cases are handled:
//  1. Same provider: AuthData matches (normal login, same OIDC sub)
//  2. Cross-provider migration: DB user has a different auth service (e.g. "gitlab")
//     but the same email. Since Mattermost only calls IsSameUser after an email
//     match, and the IdP is admin-controlled, we trust the email and allow migration.
//
// Users already on OIDC with a different sub are rejected (case 2 excludes them)
// to prevent account takeover between OIDC users.
func (p *OpenIDProvider) IsSameUser(_ request.CTX, dbUser, oAuthUser *model.User) bool {
	// Case 1: Same AuthData = same user (normal case)
	if dbUser.AuthData != nil && oAuthUser.AuthData != nil {
		if *dbUser.AuthData == *oAuthUser.AuthData {
			return true
		}
	}

	// Case 2: Cross-provider migration.
	// The DB user is on a different auth service (gitlab, email, google, etc.)
	// and we're migrating them to OIDC. Allow it — the email already matched.
	// Reject if the DB user is already on OIDC (different sub = different person).
	if dbUser.AuthService != model.ServiceOpenid {
		return true
	}

	return false
}

// mapClaimsToOIDCClaims converts JWT map claims to our structured OIDCClaims.
func mapClaimsToOIDCClaims(claims jwt.MapClaims) *OIDCClaims {
	oidcClaims := &OIDCClaims{}

	if sub, ok := claims["sub"].(string); ok {
		oidcClaims.Sub = sub
	}
	if email, ok := claims["email"].(string); ok {
		oidcClaims.Email = email
	}
	if emailVerified, ok := claims["email_verified"].(bool); ok {
		oidcClaims.EmailVerified = emailVerified
	}
	if preferredUsername, ok := claims["preferred_username"].(string); ok {
		oidcClaims.PreferredUsername = preferredUsername
	}
	if givenName, ok := claims["given_name"].(string); ok {
		oidcClaims.GivenName = givenName
	}
	if familyName, ok := claims["family_name"].(string); ok {
		oidcClaims.FamilyName = familyName
	}
	if name, ok := claims["name"].(string); ok {
		oidcClaims.Name = name
	}
	if picture, ok := claims["picture"].(string); ok {
		oidcClaims.Picture = picture
	}

	return oidcClaims
}

// OIDCClaimsFromJSON parses OIDC claims from a JSON byte slice.
// Useful for testing and direct JSON parsing.
func OIDCClaimsFromJSON(data []byte) (*OIDCClaims, error) {
	var claims OIDCClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// NormalizeEmail ensures the email is lowercase.
// Used internally for consistent email comparison.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
