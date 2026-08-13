# Deployment Guide

Building the Mattermost server binary or Docker image with the OIDC patch applied. How you run the result — systemd on a VM, Docker, Kubernetes, whatever — is up to you.

## Option 1: Local Build

Build Mattermost locally with the OIDC module. There is no Mattermost fork — the integration is a `git apply` against an upstream checkout.

```bash
# Clone upstream Mattermost at the version the patch targets
git clone --depth 1 --branch v11.8.5 https://github.com/mattermost/mattermost.git

# Clone the OIDC module as a sibling (not inside)
git clone https://github.com/toowoxx/mattermost-oidc.git

# Apply the OIDC patch (go.mod require/replace, main.go import,
# user.go email-migration change, and client.go license-gate bypass)
cd mattermost
git apply ../mattermost-oidc/patches/mattermost-v11.8.5.patch

# (Optional) AGPL-only build: remove enterprise and strip its import
rm -rf server/enterprise
sed -i '/Enterprise Imports/d; /github.com\/mattermost\/mattermost\/server\/v8\/enterprise/d' \
    server/cmd/mattermost/main.go

# Set up a go.work at the common parent so the server resolves
# mattermost-oidc locally (Mattermost doesn't publish server/v8 via the proxy).
# server/public must be included too: the server references in-tree public
# symbols newer than the tagged release, so omitting it breaks the build.
cd ..
cat > go.work <<'EOF'
go 1.26.3

use (
    ./mattermost/server
    ./mattermost/server/public
    ./mattermost-oidc
)
EOF

# Build
cd mattermost/server
GOPRIVATE='github.com/mattermost/*' make build-linux-amd64

# The binary is at ./bin/mattermost
```

## Option 2: Docker Build

The `Dockerfile` at the root of this repository does the same thing inside a container: clones upstream at the version in `MM_VERSION`, applies the patch, strips the enterprise directory, and builds a team-edition binary on Alpine.

```bash
docker build -t your-registry/mattermost-oidc:11.8.5 .
docker push your-registry/mattermost-oidc:11.8.5
```

The image exposes `8065` and runs `mattermost server` as a non-root user.
