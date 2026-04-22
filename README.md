# OIDC SSO Provider for Mattermost

A generic OpenID Connect (OIDC) SSO provider for Mattermost. Any OIDC-compliant IdP should work; we have only verified it against Entra ID (Azure AD).

## Features

- OIDC Discovery: authorization, token, and userinfo endpoints resolved from `.well-known/openid-configuration`
- Account linking: existing Mattermost accounts with a matching email are linked to OIDC on first login
- Attribute sync on each login (via Mattermost's OAuth flow)
- Delivered as a Go module plus a small patch against upstream Mattermost — no fork

## Compatibility

- Mattermost v10.11.10 (the version the current patch targets)
- Go 1.24.6+

## Quick Start

### 1. Clone this repository

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

### 3. Apply the patch to upstream Mattermost

There is no Mattermost fork — the integration is a `git apply` against an upstream checkout. Clone it as a sibling of this repository:

```bash
git clone --depth 1 --branch v10.11.10 https://github.com/mattermost/mattermost.git ../mattermost
```

Apply the OIDC patch. It adds the `go.mod` `require`/`replace`, the `main.go` blank import, removes the email-user guard in `user.go`, and opens the OpenID frontend props without a license check:

```bash
cd ../mattermost && git apply ../mattermost-oidc/patches/mattermost-v10.11.10.patch
```

(Optional) For an AGPL-only build, remove the enterprise directory and strip its import:

```bash
rm -rf server/enterprise
sed -i '/Enterprise Imports/d; /github.com\/mattermost\/mattermost\/server\/v8\/enterprise/d' \
  server/cmd/mattermost/main.go
```

Create a `go.work` in the common parent so the server resolves `mattermost-oidc` locally:

```bash
cd ..
cat > go.work <<'EOF'
go 1.24.6

use (
    ./mattermost/server
    ./mattermost-oidc
)
EOF
```

**Note:** Mattermost doesn't publish `server/v8` to the Go module proxy. Set `GOPRIVATE=github.com/mattermost/*` when building.

### 4. Configure Mattermost

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

### 5. Build and run

```bash
cd mattermost/server
make build
./bin/mattermost server
```

See [docs/deployment-guide.md](docs/deployment-guide.md) for the Docker build.

## Configuration Reference

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `Enable` | bool | `false` | Enable OIDC authentication |
| `Id` | string | `""` | OAuth client ID |
| `Secret` | string | `""` | OAuth client secret |
| `DiscoveryEndpoint` | string | `""` | OIDC discovery URL. When set, `AuthEndpoint`/`TokenEndpoint`/`UserAPIEndpoint` are resolved from it. |
| `AuthEndpoint` | string | `""` | Authorization endpoint (ignored if `DiscoveryEndpoint` is set) |
| `TokenEndpoint` | string | `""` | Token endpoint (ignored if `DiscoveryEndpoint` is set) |
| `UserAPIEndpoint` | string | `""` | UserInfo endpoint (ignored if `DiscoveryEndpoint` is set) |
| `Scope` | string | `"openid email profile"` | OAuth scopes to request |
| `ButtonText` | string | `"OpenID Connect"` | Login button text |
| `ButtonColor` | string | `"#145DBF"` | Login button color |

## OIDC Claims Mapping

| OIDC Claim | Mattermost Field | Notes |
|------------|------------------|-------|
| `sub` | `AuthData` | Unique user identifier (required) |
| `email` | `Email` | Required, lowercased |
| `email_verified` | `EmailVerified` | Passed through from the IdP |
| `preferred_username` | `Username` | Sanitized via `CleanUsername`; falls back to the local part of `email` |
| `given_name` | `FirstName` | |
| `family_name` | `LastName` | |
| `name` | `FirstName` + `LastName` | Used when `given_name`/`family_name` are absent; split on the first space |

## Identity Provider Setup

Entra ID (Azure AD) is what we use:

1. Register a new application in Entra ID.
2. Set the redirect URI to `https://your-mattermost.com/signup/openid/complete`.
3. Create a client secret.
4. Discovery endpoint: `https://login.microsoftonline.com/{tenant}/v2.0/.well-known/openid-configuration`.

Other OIDC-compliant IdPs should work the same way — point at their discovery endpoint and supply client ID/secret. We just haven't run them.

## Account Linking

With the patch applied, `IsSameUser` allows an existing Mattermost user (any non-OIDC auth service) to be linked to their OIDC account on first login if the email matches. This is always-on — there is no toggle.

**Verified cases:** GitLab → OIDC and password/email auth → OIDC.

Other source services (google, office365, saml, ldap) are handled symmetrically in code (`openid/openid.go`), but we have not exercised those paths in production.

To disable linking, revert the `server/channels/app/user.go` hunk in the patch. The `main.go`, `client.go`, and `go.mod` hunks are required regardless.

## Security

- State parameter validation is handled by Mattermost's OAuth core (timestamp, nonce, signature; one-time use; 30-minute expiry).
- `sub` is used as `AuthData` — a stable identifier that does not change when the user's email or username changes.
- HTTPS is required for OIDC endpoints in production.

## License

AGPL-3.0 — see [LICENSE](LICENSE).
