# Build stage
FROM golang:1.24.6-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Clone Mattermost (using your fork with the OIDC import)
# In CI, this would be a checkout step instead
ARG MATTERMOST_REPO=https://github.com/toowoxx/mattermost.git
ARG MATTERMOST_BRANCH=release-11.0
RUN git clone --depth 1 --branch ${MATTERMOST_BRANCH} ${MATTERMOST_REPO} mattermost

# Copy the OIDC module
COPY . mattermost-oidc/

# Build the server
WORKDIR /build/mattermost/server
RUN make config-reset
RUN make build-linux-amd64

# Runtime stage - use official Mattermost image as base
FROM mattermost/mattermost-enterprise-edition:11.0

# Copy the custom-built binary with OIDC support
COPY --from=builder /build/mattermost/server/bin/mattermost /mattermost/bin/mattermost

# Default environment variables for OIDC (can be overridden)
ENV MM_OPENIDSETTINGS_ENABLE=false \
    MM_OPENIDSETTINGS_SCOPE="openid email profile" \
    MM_OPENIDSETTINGS_BUTTONTEXT="Login with SSO" \
    MM_OPENIDSETTINGS_BUTTONCOLOR="#145DBF"

# Expose ports
EXPOSE 8065 8067 8074 8075

# Use the default entrypoint from the base image
