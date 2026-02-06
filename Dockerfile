# Build stage - compile Mattermost server with OIDC support
FROM golang:1.24.6-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

ARG MATTERMOST_REPO=https://github.com/mattermost/mattermost.git
ARG MATTERMOST_BRANCH=v10.11.10

WORKDIR /build

# Clone Mattermost (AGPL source only)
RUN git clone --depth 1 --branch ${MATTERMOST_BRANCH} ${MATTERMOST_REPO} mattermost

# Copy the OIDC module
COPY . mattermost-oidc/

# Remove enterprise code and apply the OIDC patch
RUN rm -rf /build/mattermost/server/enterprise \
    && cd /build/mattermost \
    && git apply /build/mattermost-oidc/patches/mattermost-v10.11.10.patch \
    && sed -i '/Enterprise Imports/d; /github.com\/mattermost\/mattermost\/server\/v8\/enterprise/d' \
    server/cmd/mattermost/main.go

# Set up Go workspace so the server resolves mattermost-oidc locally
RUN cat > /build/go.work <<'GOWORK'
go 1.24.6

use (
    ./mattermost/server
    ./mattermost-oidc
)
GOWORK

# Build the server binary
WORKDIR /build/mattermost/server
ENV GOPRIVATE=github.com/mattermost/* \
    GONOSUMDB=github.com/mattermost/* \
    GONOPROXY=github.com/mattermost/*
RUN go build -o bin/mattermost ./cmd/mattermost

# Runtime stage - use official Mattermost team edition image as base
FROM mattermost/mattermost-team-edition:10.11.10

# Copy the custom-built binary with OIDC support
COPY --from=builder /build/mattermost/server/bin/mattermost /mattermost/bin/mattermost
