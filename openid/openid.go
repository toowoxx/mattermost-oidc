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
// If DiscoveryEndpoint is configured, the authorization, token, and userinfo
// endpoints are resolved from the OIDC discovery document automatically.
func (p *OpenIDProvider) GetSSOSettings(_ request.CTX, config *model.Config, service string) (*model.SSOSettings, error) {
	sso := config.OpenIdSettings
	if sso.DiscoveryEndpoint != nil && *sso.DiscoveryEndpoint != "" {
		doc, err := GetDiscovery(*sso.DiscoveryEndpoint)
		if err != nil {
			return nil, err
		}
		sso.AuthEndpoint = &doc.AuthorizationEndpoint
		sso.TokenEndpoint = &doc.TokenEndpoint
		sso.UserAPIEndpoint = &doc.UserInfoEndpoint
	}
	return &sso, nil
}

// GetUserFromIdToken is not implemented. We always return (nil, nil) to let
// Mattermost core fall back to the UserInfo endpoint.
//
// Mattermost core passes the raw ID token from the token endpoint without any
// validation (no signature, iss, or aud checks). While the token is received
// over TLS, we cannot verify it properly without access to the IdP's JWKS and
// expected issuer/audience values — which are not available in this method's
// interface. Rather than parse unverified JWTs, we skip this optimization and
// rely on the authenticated UserInfo endpoint instead.
func (p *OpenIDProvider) GetUserFromIdToken(_ request.CTX, _ string) (*model.User, error) {
	return nil, nil
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
