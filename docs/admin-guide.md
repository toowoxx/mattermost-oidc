# OIDC SSO Administration Guide

This guide covers the configuration and administration of the Mattermost OIDC SSO provider.

## Prerequisites

Before configuring OIDC SSO, ensure you have:

1. A Mattermost server built with the mattermost-oidc module
2. An OIDC-compliant identity provider (IdP) configured with a client application
3. The following information from your IdP:
   - Client ID
   - Client Secret
   - Discovery endpoint URL (or individual endpoint URLs)

## Identity Provider Configuration

### Keycloak

1. Create a new client in your realm
2. Set **Client Protocol** to `openid-connect`
3. Set **Access Type** to `confidential`
4. Add valid redirect URIs: `https://your-mattermost.com/signup/openid/complete`
5. Copy the Client ID and Secret

Discovery endpoint: `https://keycloak.example.com/realms/{realm}/.well-known/openid-configuration`

### Auth0

1. Create a new Regular Web Application
2. Configure allowed callback URLs: `https://your-mattermost.com/signup/openid/complete`
3. Copy the Client ID and Secret from Settings

Discovery endpoint: `https://{tenant}.auth0.com/.well-known/openid-configuration`

### Azure AD

1. Register a new application in Azure AD
2. Set redirect URI: `https://your-mattermost.com/signup/openid/complete`
3. Create a client secret
4. Copy the Application (client) ID and secret

Discovery endpoint: `https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration`

### GitLab

1. Create a new application in Admin Area > Applications
2. Set redirect URI: `https://your-mattermost.com/signup/openid/complete`
3. Select scopes: `openid`, `email`, `profile`
4. Copy the Application ID and Secret

Discovery endpoint: `https://gitlab.example.com/.well-known/openid-configuration`

### Google

1. Create OAuth 2.0 credentials in Google Cloud Console
2. Add authorized redirect URI: `https://your-mattermost.com/signup/openid/complete`
3. Copy the Client ID and Secret

Discovery endpoint: `https://accounts.google.com/.well-known/openid-configuration`

## Mattermost Configuration

### Using config.json

```json
{
  "OpenIdSettings": {
    "Enable": true,
    "Id": "your-client-id",
    "Secret": "your-client-secret",
    "DiscoveryEndpoint": "https://your-idp.com/.well-known/openid-configuration",
    "Scope": "openid email profile",
    "ButtonText": "Login with SSO",
    "ButtonColor": "#0058CC"
  }
}
```

### Using Environment Variables

```bash
# Required settings
MM_OPENIDSETTINGS_ENABLE=true
MM_OPENIDSETTINGS_ID=your-client-id
MM_OPENIDSETTINGS_SECRET=your-client-secret

# Recommended: Use discovery endpoint for automatic configuration
MM_OPENIDSETTINGS_DISCOVERYENDPOINT=https://your-idp.com/.well-known/openid-configuration

# Optional: Manual endpoint configuration (if not using discovery)
MM_OPENIDSETTINGS_AUTHENDPOINT=https://your-idp.com/authorize
MM_OPENIDSETTINGS_TOKENENDPOINT=https://your-idp.com/token
MM_OPENIDSETTINGS_USERAPIENDPOINT=https://your-idp.com/userinfo

# Customization
MM_OPENIDSETTINGS_SCOPE="openid email profile"
MM_OPENIDSETTINGS_BUTTONTEXT="Login with SSO"
MM_OPENIDSETTINGS_BUTTONCOLOR="#0058CC"
```

### Using System Console

1. Go to **System Console > Authentication > OpenID Connect**
2. Enable OpenID Connect authentication
3. Enter your Client ID and Secret
4. Enter the Discovery Endpoint URL
5. Customize the button text and color as needed
6. Save changes

## Configuration Options

### Enable
- **Type**: Boolean
- **Default**: `false`
- **Description**: Enable or disable OIDC authentication

### Id (Client ID)
- **Type**: String
- **Required**: Yes
- **Description**: The OAuth 2.0 client identifier issued by your IdP

### Secret (Client Secret)
- **Type**: String
- **Required**: Yes
- **Description**: The OAuth 2.0 client secret. This value is encrypted at rest.

### DiscoveryEndpoint
- **Type**: String
- **Required**: Recommended
- **Description**: The OIDC discovery endpoint URL. When provided, the authorization, token, and userinfo endpoints are automatically configured.

### AuthEndpoint
- **Type**: String
- **Required**: Only if not using discovery
- **Description**: The OAuth 2.0 authorization endpoint URL

### TokenEndpoint
- **Type**: String
- **Required**: Only if not using discovery
- **Description**: The OAuth 2.0 token endpoint URL

### UserAPIEndpoint
- **Type**: String
- **Required**: Only if not using discovery
- **Description**: The OIDC UserInfo endpoint URL

### Scope
- **Type**: String
- **Default**: `"openid email profile"`
- **Description**: The OAuth scopes to request. Must include `openid` and `email` at minimum.

### ButtonText
- **Type**: String
- **Default**: `"OpenID Connect"`
- **Description**: The text displayed on the login button

### ButtonColor
- **Type**: String
- **Default**: `"#145DBF"`
- **Description**: The background color of the login button (hex color code)

## Scopes Reference

The following scopes are commonly used:

| Scope | Claims Provided |
|-------|-----------------|
| `openid` | `sub` (required) |
| `email` | `email`, `email_verified` |
| `profile` | `name`, `given_name`, `family_name`, `preferred_username` |

Minimum required scope: `openid email`
Recommended scope: `openid email profile`

## User Attribute Mapping

OIDC claims are mapped to Mattermost user attributes as follows:

| OIDC Claim | Mattermost Attribute | Notes |
|------------|---------------------|-------|
| `sub` | `AuthData` | Unique, stable identifier. Never changes. |
| `email` | `Email` | Automatically lowercased. Must be unique. |
| `preferred_username` | `Username` | Sanitized to meet Mattermost requirements |
| `given_name` | `FirstName` | User's first name |
| `family_name` | `LastName` | User's last name |
| `name` | `FirstName` + `LastName` | Used if given/family names unavailable |

### Username Handling

The username is determined in this order:
1. `preferred_username` claim (if present)
2. Local part of email address (part before @)

The username is then sanitized:
- Converted to lowercase
- Invalid characters replaced with underscores
- Truncated to maximum length
- Made unique by appending numbers if needed

## Security Considerations

### State Parameter

The Mattermost OAuth implementation includes built-in state parameter validation:
- 3-factor validation (timestamp, nonce, signature)
- One-time use tokens
- 30-minute expiry

### HTTPS Requirement

All OIDC endpoints must use HTTPS in production. HTTP endpoints are only allowed for local development.

### Client Secret Protection

- The client secret is encrypted at rest in the Mattermost database
- Never commit secrets to version control
- Use environment variables in containerized deployments

## Troubleshooting

### Common Issues

**"Invalid redirect URI" error from IdP**
- Verify the redirect URI is exactly: `https://your-mattermost.com/signup/openid/complete`
- Check for trailing slashes or protocol mismatches

**"Unable to get user from ID token" error**
- Ensure the `email` scope is included
- Verify the IdP returns the `email` claim

**User attributes not updating**
- Check that the `profile` scope is included
- Verify the IdP returns the expected claims

**Login button not appearing**
- Verify `Enable` is set to `true`
- Check that Client ID and Secret are configured
- Verify at least one endpoint is configured (discovery or manual)

### Debug Logging

Enable debug logging to troubleshoot authentication issues:

```json
{
  "LogSettings": {
    "EnableConsole": true,
    "ConsoleLevel": "DEBUG"
  }
}
```

Look for log entries containing `oauth` or `openid` to trace the authentication flow.

## High Availability Considerations

When running Mattermost in a clustered configuration:

1. Ensure all nodes have the same OIDC configuration
2. The state token is stored in the database, so it works across nodes
3. Consider session affinity if using load balancers (optional, but recommended)

## Backup and Recovery

The OIDC configuration is stored in the Mattermost config.json or database. Ensure your backup strategy includes:

1. The config.json file (if using file-based configuration)
2. The Mattermost database (always)

Note: The client secret is encrypted in the database and sanitized in config backups.
