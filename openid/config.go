// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

package openid

// OIDCConfig holds OIDC-specific configuration settings.
// These settings extend the base Mattermost OpenIdSettings.
type OIDCConfig struct {
	// EnablePKCE enables PKCE (Proof Key for Code Exchange) for the authorization code flow.
	// Recommended for security. Default: true
	EnablePKCE bool `json:"EnablePKCE"`

	// EnableAutoMigration enables automatic migration of users from other auth services
	// when they log in via OIDC with a matching email address.
	// Default: false (opt-in for safety)
	EnableAutoMigration bool `json:"EnableAutoMigration"`

	// MigrationSources lists the auth services from which users can be migrated.
	// Only users with AuthService matching one of these values will be migrated.
	// Valid values: "gitlab", "email", "google", "office365", "saml", "ldap"
	// Example: ["gitlab", "email"]
	MigrationSources []string `json:"MigrationSources"`

	// RequireEmailVerified requires the OIDC email_verified claim to be true.
	// If the claim is false or missing, login will be rejected.
	// Default: false
	RequireEmailVerified bool `json:"RequireEmailVerified"`
}

// DefaultOIDCConfig returns the default OIDC configuration.
func DefaultOIDCConfig() *OIDCConfig {
	return &OIDCConfig{
		EnablePKCE:           true,
		EnableAutoMigration:  false,
		MigrationSources:     []string{},
		RequireEmailVerified: false,
	}
}

// IsMigrationSourceAllowed checks if a given auth service is in the allowed migration sources.
func (c *OIDCConfig) IsMigrationSourceAllowed(authService string) bool {
	for _, source := range c.MigrationSources {
		if source == authService {
			return true
		}
	}
	return false
}

// ValidAuthServices lists all valid Mattermost auth service identifiers.
var ValidAuthServices = []string{
	"gitlab",
	"email",
	"google",
	"office365",
	"saml",
	"ldap",
	"openid",
}

// IsValidAuthService checks if a given auth service identifier is valid.
func IsValidAuthService(service string) bool {
	for _, valid := range ValidAuthServices {
		if service == valid {
			return true
		}
	}
	return false
}
