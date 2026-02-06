# Mattermost OIDC SSO Provider

A generic OpenID Connect (OIDC) SSO provider for Mattermost that works with any OIDC-compliant identity provider.

## Features

- **Universal OIDC Support**: Works with Keycloak, Auth0, GitLab, Azure AD, Google, Okta, and any OIDC-compliant provider
- **PKCE Support**: Proof Key for Code Exchange for enhanced security
- **OIDC Discovery**: Automatic endpoint configuration via `.well-known/openid-configuration`
- **User Migration**: Optional migration from GitLab SSO or password auth (email-based linking)
- **Attribute Sync**: Automatic user attribute synchronization on each login
- **Minimal Fork Changes**: Only 3 lines changed in the Mattermost fork

## Compatibility

- Mattermost v10.11+
- Mattermost v11.0+
- Go 1.24.6+

## Quick Start

### 1. Clone or download this repository

```bash
git clone https://github.com/toowoxx/mattermost-oidc.git
```

### 2. Development with Nix (recommended)

```bash
cd mattermost-oidc
nix develop  # Sets up Go 1.24 and GOPRIVATE automatically
go test ./...
go build ./...
```

### 3. Add to Mattermost fork

Add to `server/go.mod`:
```go
require github.com/toowoxx/mattermost-oidc v0.0.0

replace github.com/toowoxx/mattermost-oidc => ../mattermost-oidc
```

**Important:** Mattermost doesn't publish `server/v8` to the Go module proxy. Set `GOPRIVATE=github.com/mattermost/*` when building.

Add to `server/cmd/mattermost/main.go`:
```go
import (
    // ... existing imports ...
    _ "github.com/toowoxx/mattermost-oidc/openid"
)
```

### 3. Configure Mattermost

In `config.json` or via environment variables:

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

Or using environment variables:
```bash
MM_OPENIDSETTINGS_ENABLE=true
MM_OPENIDSETTINGS_ID=your-client-id
MM_OPENIDSETTINGS_SECRET=your-client-secret
MM_OPENIDSETTINGS_DISCOVERYENDPOINT=https://your-idp.com/.well-known/openid-configuration
```

### 4. Build and run

```bash
cd mattermost/server
make build
./bin/mattermost server
```

## Documentation

- [Admin Guide](docs/admin-guide.md) - Configuration reference and setup instructions
- [Migration Guide](docs/migration-guide.md) - Migrating from GitLab SSO or password auth
- [Deployment Guide](docs/deployment-guide.md) - Docker and production deployment

## Configuration Reference

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `Enable` | bool | `false` | Enable OIDC authentication |
| `Id` | string | `""` | OAuth client ID |
| `Secret` | string | `""` | OAuth client secret |
| `DiscoveryEndpoint` | string | `""` | OIDC discovery URL (recommended) |
| `AuthEndpoint` | string | `""` | Authorization endpoint (if not using discovery) |
| `TokenEndpoint` | string | `""` | Token endpoint (if not using discovery) |
| `UserAPIEndpoint` | string | `""` | UserInfo endpoint (if not using discovery) |
| `Scope` | string | `"openid email profile"` | OAuth scopes to request |
| `ButtonText` | string | `"OpenID Connect"` | Login button text |
| `ButtonColor` | string | `"#145DBF"` | Login button color |

## OIDC Claims Mapping

| OIDC Claim | Mattermost Field | Notes |
|------------|------------------|-------|
| `sub` | `AuthData` | Unique user identifier (required) |
| `email` | `Email` | User email (required, lowercased) |
| `preferred_username` | `Username` | Cleaned via `CleanUsername` |
| `given_name` | `FirstName` | First name |
| `family_name` | `LastName` | Last name |
| `name` | `FirstName` + `LastName` | Split if structured names unavailable |

## Security

- **PKCE**: Enabled by default for authorization code flow protection
- **State Validation**: Built into Mattermost OAuth core (3-factor validation)
- **Stable Identifiers**: Uses OIDC `sub` claim (never email/username which can change)
- **HTTPS**: All OIDC endpoints must use HTTPS

## License

AGPL-3.0 - See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
