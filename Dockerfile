ARG MM_VERSION=10.11.10
ARG MM_CLONE_URL=https://github.com/mattermost/mattermost.git

# Build stage - compile Mattermost server with OIDC support
FROM golang:1.24.6-alpine AS server-builder

RUN apk add --no-cache git gcc musl-dev

ARG MM_VERSION
ARG MM_CLONE_URL

WORKDIR /build

RUN git clone --depth 1 --branch v${MM_VERSION} ${MM_CLONE_URL} mattermost

COPY . mattermost-oidc/

RUN rm -rf /build/mattermost/server/enterprise \
    && cd /build/mattermost \
    && git apply /build/mattermost-oidc/patches/mattermost-v10.11.10.patch \
    && sed -i '/Enterprise Imports/d; /github.com\/mattermost\/mattermost\/server\/v8\/enterprise/d' \
    server/cmd/mattermost/main.go

RUN cat > /build/go.work <<'GOWORK'
go 1.24.6

use (
    ./mattermost/server
    ./mattermost-oidc
)
GOWORK

WORKDIR /build/mattermost/server
ENV GOPRIVATE=github.com/mattermost/* \
    GONOSUMDB=github.com/mattermost/* \
    GONOPROXY=github.com/mattermost/*
RUN go build -ldflags "-X github.com/mattermost/mattermost/server/public/model.BuildNumber=${MM_VERSION}" \
    -o bin/mattermost ./cmd/mattermost

# Runtime stage
FROM alpine:3.18

ENV PATH="/mattermost/bin:${PATH}" \
    MM_INSTALL_TYPE=docker

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /mattermost/bin /mattermost/data /mattermost/plugins /mattermost/client/plugins /mattermost/config /mattermost/logs \
    && addgroup -S mattermost \
    && adduser -S -G mattermost -h /mattermost mattermost

COPY --from=server-builder /build/mattermost/server/bin/mattermost /mattermost/bin/mattermost
COPY --from=server-builder /build/mattermost/config/config.json /mattermost/config/config.json
COPY --from=server-builder /build/mattermost/server/i18n /mattermost/i18n
COPY --from=server-builder /build/mattermost/server/templates /mattermost/templates
COPY --from=server-builder /build/mattermost/server/fonts /mattermost/fonts
RUN chown -R mattermost:mattermost /mattermost

USER mattermost

WORKDIR /mattermost

EXPOSE 8065

CMD ["mattermost", "server"]
