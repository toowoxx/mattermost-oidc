# Migration Guide

This guide explains how to migrate existing users from other authentication methods (GitLab SSO, password/email auth) to OIDC authentication.

## Overview

The mattermost-oidc provider supports optional automatic migration of existing users based on email address matching. When enabled, users logging in via OIDC for the first time will have their existing accounts migrated rather than creating duplicate accounts.

## Migration Methods

### Method 1: Automatic Migration (Recommended)

Automatic migration links existing user accounts to OIDC based on matching email addresses. This is the safest and most seamless approach for users.

**How it works:**
1. User clicks "Login with OIDC"
2. User authenticates with the IdP
3. System checks if a user exists with the OIDC `sub` claim
4. If not found and migration is enabled, searches for user by email
5. If found with an allowed auth service, updates the user's auth to OIDC
6. User is logged in with their existing account

**Requirements:**
- User's email in Mattermost must match the email from the IdP
- User's current auth service must be in the allowed migration sources
- Auto-migration must be explicitly enabled (opt-in)

### Method 2: Manual Migration

For more control, administrators can manually migrate users via the Mattermost API or database.

**Via mmctl:**
```bash
# Get user details
mmctl user search john.doe@example.com

# Update user auth (requires direct database access or custom script)
```

**Via Database:**
```sql
-- Update a specific user's auth service and data
UPDATE Users
SET AuthService = 'openid',
    AuthData = 'oidc-sub-claim-value'
WHERE Email = 'john.doe@example.com';
```

**Warning:** Manual database updates bypass validation. Ensure the OIDC `sub` claim is correct.

## Configuration

### Enabling Auto-Migration

Auto-migration is configured via plugin settings. Add to your `config.json`:

```json
{
  "PluginSettings": {
    "Plugins": {
      "com.toowoxx.mattermost-oidc": {
        "EnableAutoMigration": true,
        "MigrationSources": ["gitlab", "email"]
      }
    }
  }
}
```

Or via environment variables:
```bash
MM_PLUGINSETTINGS_PLUGINS_COM_TOOWOXX_MATTERMOST_OIDC_ENABLEAUTOMIGRATION=true
MM_PLUGINSETTINGS_PLUGINS_COM_TOOWOXX_MATTERMOST_OIDC_MIGRATIONSOURCES='["gitlab", "email"]'
```

### Configuration Options

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `EnableAutoMigration` | bool | `false` | Enable automatic user migration |
| `MigrationSources` | []string | `[]` | Auth services to migrate from |

### Valid Migration Sources

- `gitlab` - GitLab OAuth/OIDC
- `email` - Password/email authentication
- `google` - Google OAuth
- `office365` - Microsoft Office 365
- `saml` - SAML authentication
- `ldap` - LDAP authentication

## Migration Scenarios

### Scenario 1: GitLab SSO to Generic OIDC

**Use case:** You're switching from GitLab's OAuth to a dedicated IdP (e.g., Keycloak).

**Configuration:**
```json
{
  "PluginSettings": {
    "Plugins": {
      "com.toowoxx.mattermost-oidc": {
        "EnableAutoMigration": true,
        "MigrationSources": ["gitlab"]
      }
    }
  }
}
```

**Process:**
1. Set up the new OIDC provider in Mattermost
2. Ensure all users exist in the new IdP with matching emails
3. Enable auto-migration
4. Users log in via the new OIDC button
5. Their GitLab SSO accounts are automatically migrated
6. After migration, optionally disable GitLab OAuth

### Scenario 2: Password Auth to SSO

**Use case:** You want to move users from password authentication to SSO.

**Configuration:**
```json
{
  "PluginSettings": {
    "Plugins": {
      "com.toowoxx.mattermost-oidc": {
        "EnableAutoMigration": true,
        "MigrationSources": ["email"]
      }
    }
  }
}
```

**Process:**
1. Configure OIDC authentication
2. Create users in your IdP with matching emails
3. Enable auto-migration for email auth
4. Users log in via OIDC
5. Their password accounts are migrated to OIDC
6. Optionally disable email/password authentication

### Scenario 3: Multiple Auth Sources

**Use case:** You have users on both GitLab SSO and password auth.

**Configuration:**
```json
{
  "PluginSettings": {
    "Plugins": {
      "com.toowoxx.mattermost-oidc": {
        "EnableAutoMigration": true,
        "MigrationSources": ["gitlab", "email"]
      }
    }
  }
}
```

## Pre-Migration Checklist

Before enabling migration:

- [ ] Verify all users exist in the new IdP
- [ ] Confirm email addresses match between Mattermost and IdP
- [ ] Test with a few pilot users first
- [ ] Communicate the change to users
- [ ] Plan for users who may have different emails
- [ ] Back up the Mattermost database
- [ ] Document the rollback procedure

## Post-Migration Tasks

After migration is complete:

1. **Verify migrations:**
   ```sql
   SELECT COUNT(*) FROM Users WHERE AuthService = 'openid';
   ```

2. **Check for unmigrated users:**
   ```sql
   SELECT Email, AuthService FROM Users
   WHERE AuthService IN ('gitlab', 'email')
   ORDER BY AuthService;
   ```

3. **Disable old auth methods** (optional):
   - GitLab: Set `GitLabSettings.Enable` to `false`
   - Email: Set `EmailSettings.EnableSignUpWithEmail` to `false`

4. **Disable auto-migration:**
   ```json
   {
     "PluginSettings": {
       "Plugins": {
         "com.toowoxx.mattermost-oidc": {
           "EnableAutoMigration": false
         }
       }
     }
   }
   ```

## Handling Edge Cases

### Email Mismatch

If a user's email in Mattermost doesn't match the IdP:

1. Update the email in Mattermost first:
   ```bash
   mmctl user email old@example.com new@example.com
   ```
2. Then have the user log in via OIDC

### Duplicate Accounts

If migration creates a duplicate account:

1. Identify the duplicate:
   ```sql
   SELECT Id, Email, Username, AuthService, AuthData
   FROM Users
   WHERE Email = 'user@example.com';
   ```

2. Merge or delete the duplicate account using mmctl or the System Console

### Users Without IdP Accounts

For users who don't have IdP accounts:

1. Create accounts in the IdP, or
2. Keep email/password auth enabled for these users, or
3. Migrate them manually after creating IdP accounts

## Security Considerations

### During Migration

- Both old and new auth methods work simultaneously
- Users can log in with either method until migrated
- After migration, only OIDC works for that user

### Email Verification

- OIDC migration trusts the IdP's email verification
- Consider enabling `RequireEmailVerified` for additional security

### Audit Trail

Log migration events for compliance:

```sql
SELECT Id, Username, Email, AuthService, UpdateAt
FROM Users
WHERE AuthService = 'openid'
  AND UpdateAt > EXTRACT(EPOCH FROM TIMESTAMP '2026-01-01')
ORDER BY UpdateAt DESC;
```

## Rollback Procedure

If you need to roll back a migration:

1. **Disable OIDC temporarily:**
   ```bash
   MM_OPENIDSETTINGS_ENABLE=false
   ```

2. **Restore user auth service:**
   ```sql
   -- Restore specific user to GitLab
   UPDATE Users
   SET AuthService = 'gitlab',
       AuthData = 'original-gitlab-id'
   WHERE Email = 'user@example.com';

   -- Restore specific user to email auth
   UPDATE Users
   SET AuthService = '',
       AuthData = NULL
   WHERE Email = 'user@example.com';
   ```

   Note: For password auth, you may need to reset the user's password.

3. **Re-enable the original auth method**

## Troubleshooting

### "User not found for migration"

- Verify the email addresses match exactly (case-insensitive)
- Check that the auth service is in `MigrationSources`

### "Migration disabled"

- Set `EnableAutoMigration` to `true`
- Ensure `MigrationSources` is not empty

### "User already has OIDC auth"

- The user has already been migrated
- No action needed

### Monitoring Migration Progress

Track migration progress with this query:

```sql
SELECT
    AuthService,
    COUNT(*) as user_count
FROM Users
WHERE DeleteAt = 0
GROUP BY AuthService
ORDER BY user_count DESC;
```
