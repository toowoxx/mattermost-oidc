// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

package openid

import (
	"github.com/mattermost/mattermost/server/public/model"
)

// MigrationResult represents the outcome of a migration attempt.
type MigrationResult struct {
	// Migrated indicates whether a user was migrated.
	Migrated bool

	// User is the migrated user (nil if not migrated).
	User *model.User

	// PreviousAuthService is the user's previous auth service (e.g., "gitlab", "email").
	PreviousAuthService string

	// Message provides details about the migration outcome.
	Message string
}

// MigrationCandidate represents a user that could potentially be migrated to OIDC.
type MigrationCandidate struct {
	// User is the existing Mattermost user.
	User *model.User

	// OIDCSub is the OIDC subject identifier to migrate to.
	OIDCSub string

	// OIDCEmail is the email from the OIDC claims.
	OIDCEmail string
}

// PrepareMigration checks if a user is eligible for migration based on the config.
// This is a helper function that can be called from application code.
//
// The actual migration must be performed at the app layer since the provider
// does not have direct access to the store. This function only validates
// whether migration should proceed.
//
// Example usage in app layer:
//
//	func (a *App) MigrateUserToOIDC(candidate *MigrationCandidate, config *OIDCConfig) error {
//	    result := openid.PrepareMigration(candidate, config)
//	    if !result.Migrated {
//	        return nil // No migration needed
//	    }
//
//	    // Update user in database
//	    user := candidate.User
//	    user.AuthService = model.ServiceOpenid
//	    user.AuthData = &candidate.OIDCSub
//	    _, err := a.Srv().Store().User().Update(rctx, user, false)
//	    return err
//	}
func PrepareMigration(candidate *MigrationCandidate, config *OIDCConfig) *MigrationResult {
	result := &MigrationResult{
		Migrated: false,
	}

	// Check if migration is enabled
	if !config.EnableAutoMigration {
		result.Message = "auto-migration is disabled"
		return result
	}

	// Validate candidate
	if candidate == nil || candidate.User == nil {
		result.Message = "no user to migrate"
		return result
	}

	user := candidate.User

	// Don't migrate if user is already using OIDC
	if user.AuthService == model.ServiceOpenid {
		result.Message = "user is already using OIDC"
		return result
	}

	// Check if the user's current auth service is in the allowed migration sources
	if !config.IsMigrationSourceAllowed(user.AuthService) {
		result.Message = "user's auth service is not in allowed migration sources"
		return result
	}

	// Verify email match (case-insensitive)
	if NormalizeEmail(user.Email) != NormalizeEmail(candidate.OIDCEmail) {
		result.Message = "email addresses do not match"
		return result
	}

	// Migration is allowed
	result.Migrated = true
	result.User = user
	result.PreviousAuthService = user.AuthService
	result.Message = "user eligible for migration"

	return result
}

// ShouldAttemptMigration is a quick check to see if migration should even be attempted.
// Use this before doing expensive database lookups.
func ShouldAttemptMigration(config *OIDCConfig) bool {
	return config.EnableAutoMigration && len(config.MigrationSources) > 0
}

// BuildMigrationUpdate returns the fields that need to be updated for migration.
// This creates a partial user update suitable for the store layer.
func BuildMigrationUpdate(oidcSub string) map[string]interface{} {
	return map[string]interface{}{
		"AuthService": model.ServiceOpenid,
		"AuthData":    oidcSub,
	}
}

// MigrationNote documents the expected integration approach.
//
// Since OAuthProvider does not have access to the store, migration must be
// implemented at the app layer. The recommended approach is:
//
// 1. In the OAuth callback handler (after GetUserFromJSON returns):
//    - Check if user exists by AuthData (OIDC sub) - if yes, login normally
//    - Check if ShouldAttemptMigration(config) - if no, proceed with normal flow
//    - If migration enabled, search for user by email
//    - Use PrepareMigration to check eligibility
//    - If eligible, update user's AuthService and AuthData
//    - Proceed with login
//
// The core Mattermost OAuth flow already handles most of this in:
// - server/channels/app/oauth.go: CompleteOAuth
// - Specifically the GetUser and UpdateOAuthUserAttrs calls
//
// Migration can be added as a hook before the user lookup, or by extending
// the app layer with a custom migration check.
type MigrationNote struct{}
