// Copyright (c) 2026 Toowoxx IT GmbH
// Licensed under the AGPL-3.0 license. See LICENSE for details.

package openid

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// mockLogger implements mlog.LoggerIFace for testing
type mockLogger struct{}

func (m *mockLogger) IsLevelEnabled(level mlog.Level) bool                       { return true }
func (m *mockLogger) Trace(msg string, fields ...mlog.Field)                     {}
func (m *mockLogger) Debug(msg string, fields ...mlog.Field)                     {}
func (m *mockLogger) Info(msg string, fields ...mlog.Field)                      {}
func (m *mockLogger) Warn(msg string, fields ...mlog.Field)                      {}
func (m *mockLogger) Error(msg string, fields ...mlog.Field)                     {}
func (m *mockLogger) Critical(msg string, fields ...mlog.Field)                  {}
func (m *mockLogger) Fatal(msg string, fields ...mlog.Field)                     {}
func (m *mockLogger) Log(level mlog.Level, msg string, fields ...mlog.Field)     {}
func (m *mockLogger) LogM(levels []mlog.Level, msg string, fields ...mlog.Field) {}
func (m *mockLogger) With(fields ...mlog.Field) *mlog.Logger                     { return nil }
func (m *mockLogger) Flush() error                                               { return nil }
func (m *mockLogger) Sugar(fields ...mlog.Field) mlog.Sugar                      { return mlog.Sugar{} }
func (m *mockLogger) StdLogger(level mlog.Level) *log.Logger                     { return nil }

// Test claims parsing with all fields
func TestParseOIDCClaims_AllFields(t *testing.T) {
	jsonData := `{
		"sub": "user-123",
		"email": "test@example.com",
		"email_verified": true,
		"preferred_username": "testuser",
		"given_name": "Test",
		"family_name": "User",
		"name": "Test User",
		"picture": "https://example.com/photo.jpg"
	}`

	claims, err := ParseOIDCClaims(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to parse claims: %v", err)
	}

	if claims.Sub != "user-123" {
		t.Errorf("Expected sub 'user-123', got '%s'", claims.Sub)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("Expected email_verified to be true")
	}
	if claims.PreferredUsername != "testuser" {
		t.Errorf("Expected preferred_username 'testuser', got '%s'", claims.PreferredUsername)
	}
	if claims.GivenName != "Test" {
		t.Errorf("Expected given_name 'Test', got '%s'", claims.GivenName)
	}
	if claims.FamilyName != "User" {
		t.Errorf("Expected family_name 'User', got '%s'", claims.FamilyName)
	}
}

// Test claims validation
func TestOIDCClaims_Validate(t *testing.T) {
	tests := []struct {
		name    string
		claims  OIDCClaims
		wantErr bool
	}{
		{
			name: "valid claims",
			claims: OIDCClaims{
				Sub:   "user-123",
				Email: "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "missing sub",
			claims: OIDCClaims{
				Email: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "missing email",
			claims: OIDCClaims{
				Sub: "user-123",
			},
			wantErr: true,
		},
		{
			name:    "empty claims",
			claims:  OIDCClaims{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test conversion to Mattermost user
func TestOIDCClaims_ToUser(t *testing.T) {
	logger := &mockLogger{}

	tests := []struct {
		name              string
		claims            OIDCClaims
		wantUsername      string
		wantFirstName     string
		wantLastName      string
		wantEmail         string
		wantAuthData      string
		wantEmailVerified bool
	}{
		{
			name: "all fields present with verified email",
			claims: OIDCClaims{
				Sub:               "user-123",
				Email:             "Test@Example.com",
				EmailVerified:     true,
				PreferredUsername: "testuser",
				GivenName:         "Test",
				FamilyName:        "User",
			},
			wantUsername:      "testuser",
			wantFirstName:     "Test",
			wantLastName:      "User",
			wantEmail:         "test@example.com", // Lowercase
			wantAuthData:      "user-123",
			wantEmailVerified: true,
		},
		{
			name: "username from email, unverified",
			claims: OIDCClaims{
				Sub:   "user-456",
				Email: "john.doe@example.com",
			},
			wantUsername:      "john.doe",
			wantFirstName:     "",
			wantLastName:      "",
			wantEmail:         "john.doe@example.com",
			wantAuthData:      "user-456",
			wantEmailVerified: false,
		},
		{
			name: "name split fallback",
			claims: OIDCClaims{
				Sub:           "user-789",
				Email:         "test@example.com",
				EmailVerified: true,
				Name:          "John Doe Smith",
			},
			wantUsername:      "test",
			wantFirstName:     "John",
			wantLastName:      "Doe Smith",
			wantEmail:         "test@example.com",
			wantAuthData:      "user-789",
			wantEmailVerified: true,
		},
		{
			name: "single name",
			claims: OIDCClaims{
				Sub:           "user-abc",
				Email:         "test@example.com",
				EmailVerified: true,
				Name:          "Madonna",
			},
			wantUsername:      "test",
			wantFirstName:     "Madonna",
			wantLastName:      "",
			wantEmail:         "test@example.com",
			wantAuthData:      "user-abc",
			wantEmailVerified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tt.claims.ToUser(logger)

			if user.Username != tt.wantUsername {
				t.Errorf("Username = %s, want %s", user.Username, tt.wantUsername)
			}
			if user.FirstName != tt.wantFirstName {
				t.Errorf("FirstName = %s, want %s", user.FirstName, tt.wantFirstName)
			}
			if user.LastName != tt.wantLastName {
				t.Errorf("LastName = %s, want %s", user.LastName, tt.wantLastName)
			}
			if user.Email != tt.wantEmail {
				t.Errorf("Email = %s, want %s", user.Email, tt.wantEmail)
			}
			if user.AuthData == nil || *user.AuthData != tt.wantAuthData {
				t.Errorf("AuthData = %v, want %s", user.AuthData, tt.wantAuthData)
			}
			if user.AuthService != model.ServiceOpenid {
				t.Errorf("AuthService = %s, want %s", user.AuthService, model.ServiceOpenid)
			}
			if user.EmailVerified != tt.wantEmailVerified {
				t.Errorf("EmailVerified = %v, want %v", user.EmailVerified, tt.wantEmailVerified)
			}
		})
	}
}

// Test ValidateWithConfig
func TestOIDCClaims_ValidateWithConfig(t *testing.T) {
	tests := []struct {
		name    string
		claims  OIDCClaims
		config  *OIDCConfig
		wantErr bool
	}{
		{
			name:    "verified email, required",
			claims:  OIDCClaims{Sub: "u1", Email: "a@b.com", EmailVerified: true},
			config:  &OIDCConfig{RequireEmailVerified: true},
			wantErr: false,
		},
		{
			name:    "unverified email, required",
			claims:  OIDCClaims{Sub: "u1", Email: "a@b.com", EmailVerified: false},
			config:  &OIDCConfig{RequireEmailVerified: true},
			wantErr: true,
		},
		{
			name:    "unverified email, not required",
			claims:  OIDCClaims{Sub: "u1", Email: "a@b.com", EmailVerified: false},
			config:  &OIDCConfig{RequireEmailVerified: false},
			wantErr: false,
		},
		{
			name:    "nil config",
			claims:  OIDCClaims{Sub: "u1", Email: "a@b.com"},
			config:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.ValidateWithConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test discovery endpoint construction
func TestDiscoveryEndpointFromIssuer(t *testing.T) {
	tests := []struct {
		issuer string
		want   string
	}{
		{
			issuer: "https://keycloak.example.com/realms/main",
			want:   "https://keycloak.example.com/realms/main/.well-known/openid-configuration",
		},
		{
			issuer: "https://keycloak.example.com/realms/main/",
			want:   "https://keycloak.example.com/realms/main/.well-known/openid-configuration",
		},
		{
			issuer: "https://auth0.com",
			want:   "https://auth0.com/.well-known/openid-configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.issuer, func(t *testing.T) {
			got := DiscoveryEndpointFromIssuer(tt.issuer)
			if got != tt.want {
				t.Errorf("DiscoveryEndpointFromIssuer(%s) = %s, want %s", tt.issuer, got, tt.want)
			}
		})
	}
}

// Test OIDC config defaults
func TestDefaultOIDCConfig(t *testing.T) {
	config := DefaultOIDCConfig()

	if config.EnableAutoMigration {
		t.Error("EnableAutoMigration should default to false")
	}
	if len(config.MigrationSources) != 0 {
		t.Error("MigrationSources should default to empty")
	}
	if config.RequireEmailVerified {
		t.Error("RequireEmailVerified should default to false")
	}
}

// Test migration source check
func TestOIDCConfig_IsMigrationSourceAllowed(t *testing.T) {
	config := &OIDCConfig{
		MigrationSources: []string{"gitlab", "email"},
	}

	tests := []struct {
		service string
		want    bool
	}{
		{"gitlab", true},
		{"email", true},
		{"google", false},
		{"ldap", false},
		{"openid", false},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			got := config.IsMigrationSourceAllowed(tt.service)
			if got != tt.want {
				t.Errorf("IsMigrationSourceAllowed(%s) = %v, want %v", tt.service, got, tt.want)
			}
		})
	}
}

// Test migration preparation
func TestPrepareMigration(t *testing.T) {
	tests := []struct {
		name       string
		candidate  *MigrationCandidate
		config     *OIDCConfig
		wantMigrated bool
	}{
		{
			name: "migration disabled",
			candidate: &MigrationCandidate{
				User:      &model.User{Email: "test@example.com", AuthService: "gitlab"},
				OIDCSub:   "sub-123",
				OIDCEmail: "test@example.com",
			},
			config: &OIDCConfig{
				EnableAutoMigration: false,
			},
			wantMigrated: false,
		},
		{
			name: "successful migration",
			candidate: &MigrationCandidate{
				User:      &model.User{Email: "test@example.com", AuthService: "gitlab"},
				OIDCSub:   "sub-123",
				OIDCEmail: "test@example.com",
			},
			config: &OIDCConfig{
				EnableAutoMigration: true,
				MigrationSources:    []string{"gitlab", "email"},
			},
			wantMigrated: true,
		},
		{
			name: "auth service not allowed",
			candidate: &MigrationCandidate{
				User:      &model.User{Email: "test@example.com", AuthService: "ldap"},
				OIDCSub:   "sub-123",
				OIDCEmail: "test@example.com",
			},
			config: &OIDCConfig{
				EnableAutoMigration: true,
				MigrationSources:    []string{"gitlab", "email"},
			},
			wantMigrated: false,
		},
		{
			name: "email mismatch",
			candidate: &MigrationCandidate{
				User:      &model.User{Email: "old@example.com", AuthService: "gitlab"},
				OIDCSub:   "sub-123",
				OIDCEmail: "new@example.com",
			},
			config: &OIDCConfig{
				EnableAutoMigration: true,
				MigrationSources:    []string{"gitlab"},
			},
			wantMigrated: false,
		},
		{
			name: "already using OIDC",
			candidate: &MigrationCandidate{
				User:      &model.User{Email: "test@example.com", AuthService: model.ServiceOpenid},
				OIDCSub:   "sub-123",
				OIDCEmail: "test@example.com",
			},
			config: &OIDCConfig{
				EnableAutoMigration: true,
				MigrationSources:    []string{"gitlab"},
			},
			wantMigrated: false,
		},
		{
			name: "case insensitive email match",
			candidate: &MigrationCandidate{
				User:      &model.User{Email: "Test@EXAMPLE.com", AuthService: "gitlab"},
				OIDCSub:   "sub-123",
				OIDCEmail: "test@example.COM",
			},
			config: &OIDCConfig{
				EnableAutoMigration: true,
				MigrationSources:    []string{"gitlab"},
			},
			wantMigrated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrepareMigration(tt.candidate, tt.config)
			if result.Migrated != tt.wantMigrated {
				t.Errorf("PrepareMigration() Migrated = %v, want %v (message: %s)",
					result.Migrated, tt.wantMigrated, result.Message)
			}
		})
	}
}

// Test IsSameUser
func TestOpenIDProvider_IsSameUser(t *testing.T) {
	provider := &OpenIDProvider{}

	sub1 := "user-123"
	sub2 := "user-456"
	gitlabID := "gitlab-id-789"

	tests := []struct {
		name      string
		dbUser    *model.User
		oauthUser *model.User
		want      bool
	}{
		// Case 1: Same provider, same AuthData
		{
			name:      "same OIDC user",
			dbUser:    &model.User{AuthService: model.ServiceOpenid, AuthData: &sub1},
			oauthUser: &model.User{AuthService: model.ServiceOpenid, AuthData: &sub1},
			want:      true,
		},
		// Case 1: Same provider, different AuthData — different person
		{
			name:      "different OIDC user",
			dbUser:    &model.User{AuthService: model.ServiceOpenid, AuthData: &sub1},
			oauthUser: &model.User{AuthService: model.ServiceOpenid, AuthData: &sub2},
			want:      false,
		},
		// Case 2: Cross-provider migration from GitLab
		{
			name:      "gitlab to OIDC migration",
			dbUser:    &model.User{Email: "user@example.com", AuthService: "gitlab", AuthData: &gitlabID},
			oauthUser: &model.User{Email: "user@example.com", AuthService: model.ServiceOpenid, AuthData: &sub1},
			want:      true,
		},
		// Case 2: Cross-provider migration from Google
		{
			name:      "google to OIDC migration",
			dbUser:    &model.User{Email: "user@example.com", AuthService: "google", AuthData: &gitlabID},
			oauthUser: &model.User{Email: "user@example.com", AuthService: model.ServiceOpenid, AuthData: &sub1},
			want:      true,
		},
		// Case 2: Cross-provider migration from email/password auth
		{
			name:      "email auth to OIDC migration",
			dbUser:    &model.User{Email: "user@example.com", AuthService: "email", AuthData: &gitlabID},
			oauthUser: &model.User{Email: "user@example.com", AuthService: model.ServiceOpenid, AuthData: &sub1},
			want:      true,
		},
		// Reject: already OIDC with different sub (different person)
		{
			name:      "already OIDC different sub",
			dbUser:    &model.User{Email: "user@example.com", AuthService: model.ServiceOpenid, AuthData: &sub2},
			oauthUser: &model.User{Email: "user@example.com", AuthService: model.ServiceOpenid, AuthData: &sub1},
			want:      false,
		},
		// Case 2: Email/password user migration
		{
			name:      "email auth to OIDC migration (empty AuthService)",
			dbUser:    &model.User{Email: "user@example.com", AuthService: "", AuthData: nil},
			oauthUser: &model.User{Email: "user@example.com", AuthService: model.ServiceOpenid, AuthData: &sub1},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.IsSameUser(nil, tt.dbUser, tt.oauthUser)
			if got != tt.want {
				t.Errorf("IsSameUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test GetUserFromJSON
func TestOpenIDProvider_GetUserFromJSON(t *testing.T) {
	provider := &OpenIDProvider{}

	jsonData := `{
		"sub": "user-123",
		"email": "test@example.com",
		"preferred_username": "testuser",
		"given_name": "Test",
		"family_name": "User"
	}`

	// Create a mock context
	user, err := provider.GetUserFromJSON(nil, bytes.NewReader([]byte(jsonData)), nil)
	if err != nil {
		t.Fatalf("GetUserFromJSON failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", user.Email)
	}
	if *user.AuthData != "user-123" {
		t.Errorf("AuthData = %s, want user-123", *user.AuthData)
	}
}

// Test GetUserFromJSON with invalid JSON
func TestOpenIDProvider_GetUserFromJSON_InvalidJSON(t *testing.T) {
	provider := &OpenIDProvider{}

	_, err := provider.GetUserFromJSON(nil, strings.NewReader("invalid json"), nil)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// Test GetUserFromJSON with missing required claims
func TestOpenIDProvider_GetUserFromJSON_MissingClaims(t *testing.T) {
	provider := &OpenIDProvider{}

	// Missing sub
	jsonData := `{"email": "test@example.com"}`
	_, err := provider.GetUserFromJSON(nil, strings.NewReader(jsonData), nil)
	if err == nil {
		t.Error("Expected error for missing sub claim")
	}

	// Missing email
	jsonData = `{"sub": "user-123"}`
	_, err = provider.GetUserFromJSON(nil, strings.NewReader(jsonData), nil)
	if err == nil {
		t.Error("Expected error for missing email claim")
	}
}

// Test email normalization
func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Test@Example.com", "test@example.com"},
		{"  test@example.com  ", "test@example.com"},
		{"TEST@EXAMPLE.COM", "test@example.com"},
		{"test@example.com", "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeEmail(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeEmail(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// Test mapClaimsToOIDCClaims
func TestMapClaimsToOIDCClaims(t *testing.T) {
	jwtClaims := jwt.MapClaims{
		"sub":                "user-123",
		"email":              "test@example.com",
		"email_verified":     true,
		"preferred_username": "testuser",
		"given_name":         "Test",
		"family_name":        "User",
		"name":               "Test User",
		"picture":            "https://example.com/photo.jpg",
	}

	claims := mapClaimsToOIDCClaims(jwtClaims)

	if claims.Sub != "user-123" {
		t.Errorf("Sub = %s, want user-123", claims.Sub)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("EmailVerified should be true")
	}
	if claims.PreferredUsername != "testuser" {
		t.Errorf("PreferredUsername = %s, want testuser", claims.PreferredUsername)
	}
	if claims.GivenName != "Test" {
		t.Errorf("GivenName = %s, want Test", claims.GivenName)
	}
	if claims.FamilyName != "User" {
		t.Errorf("FamilyName = %s, want User", claims.FamilyName)
	}
	if claims.Name != "Test User" {
		t.Errorf("Name = %s, want Test User", claims.Name)
	}
	if claims.Picture != "https://example.com/photo.jpg" {
		t.Errorf("Picture = %s, want https://example.com/photo.jpg", claims.Picture)
	}
}

// Test mapClaimsToOIDCClaims handles email_verified type variants
func TestMapClaimsToOIDCClaims_EmailVerifiedTypes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string false", "false", false},
		{"float64 1", float64(1), true},
		{"float64 0", float64(0), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtClaims := jwt.MapClaims{
				"sub":            "u1",
				"email":          "a@b.com",
				"email_verified": tt.value,
			}
			claims := mapClaimsToOIDCClaims(jwtClaims)
			if claims.EmailVerified != tt.want {
				t.Errorf("EmailVerified = %v, want %v", claims.EmailVerified, tt.want)
			}
		})
	}
}

// Test ShouldAttemptMigration
func TestShouldAttemptMigration(t *testing.T) {
	tests := []struct {
		name   string
		config *OIDCConfig
		want   bool
	}{
		{
			name:   "migration disabled",
			config: &OIDCConfig{EnableAutoMigration: false},
			want:   false,
		},
		{
			name:   "migration enabled but no sources",
			config: &OIDCConfig{EnableAutoMigration: true, MigrationSources: []string{}},
			want:   false,
		},
		{
			name:   "migration enabled with sources",
			config: &OIDCConfig{EnableAutoMigration: true, MigrationSources: []string{"gitlab"}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldAttemptMigration(tt.config)
			if got != tt.want {
				t.Errorf("ShouldAttemptMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark claims parsing
func BenchmarkParseOIDCClaims(b *testing.B) {
	jsonData := `{
		"sub": "user-123",
		"email": "test@example.com",
		"email_verified": true,
		"preferred_username": "testuser",
		"given_name": "Test",
		"family_name": "User",
		"name": "Test User"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseOIDCClaims(strings.NewReader(jsonData))
	}
}
