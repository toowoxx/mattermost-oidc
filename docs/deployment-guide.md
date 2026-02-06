# Deployment Guide

This guide covers building and deploying Mattermost with the OIDC SSO provider.

## Build Options

### Option 1: Local Build

Build Mattermost locally with the OIDC module:

```bash
# Clone the Mattermost fork
git clone https://github.com/toowoxx/mattermost.git
cd mattermost

# Clone the OIDC module alongside (not inside)
cd ..
git clone https://github.com/toowoxx/mattermost-oidc.git

# Build
cd mattermost/server
make build-linux-amd64  # or make build for current platform

# The binary is at ./bin/mattermost
```

### Option 2: Docker Build

Build a custom Docker image with OIDC support.

**Dockerfile:**

```dockerfile
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
```

**Build and push:**

```bash
docker build --build-arg MM_VERSION=10.11.10 -t your-registry/mattermost-oidc:10.11.10 .
docker push your-registry/mattermost-oidc:10.11.10
```

### Option 3: GitHub Actions CI/CD

**`.github/workflows/build.yml`:**

```yaml
name: Build Mattermost with OIDC

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout OIDC module
        uses: actions/checkout@v4

      - name: Checkout Mattermost fork
        uses: actions/checkout@v4
        with:
          repository: toowoxx/mattermost
          path: mattermost
          ref: release-11.0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24.6'

      - name: Build server
        run: |
          cd mattermost/server
          make build-linux-amd64

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: ${{ github.event_name != 'pull_request' }}
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
            ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:${{ github.sha }}
```

## Deployment Methods

### Docker Compose

**`docker-compose.yml`:**

```yaml
version: '3.8'

services:
  mattermost:
    image: your-registry/mattermost-oidc:11.0
    container_name: mattermost
    restart: unless-stopped
    ports:
      - "8065:8065"
    environment:
      # Database
      MM_SQLSETTINGS_DRIVERNAME: postgres
      MM_SQLSETTINGS_DATASOURCE: postgres://mmuser:mmpassword@postgres:5432/mattermost?sslmode=disable

      # OIDC Configuration
      MM_OPENIDSETTINGS_ENABLE: "true"
      MM_OPENIDSETTINGS_ID: "${OIDC_CLIENT_ID}"
      MM_OPENIDSETTINGS_SECRET: "${OIDC_CLIENT_SECRET}"
      MM_OPENIDSETTINGS_DISCOVERYENDPOINT: "${OIDC_DISCOVERY_URL}"
      MM_OPENIDSETTINGS_SCOPE: "openid email profile"
      MM_OPENIDSETTINGS_BUTTONTEXT: "Login with SSO"
      MM_OPENIDSETTINGS_BUTTONCOLOR: "#0058CC"

      # Site URL (important for OAuth redirects)
      MM_SERVICESETTINGS_SITEURL: "https://mattermost.example.com"
    volumes:
      - mattermost-data:/mattermost/data
      - mattermost-logs:/mattermost/logs
      - mattermost-config:/mattermost/config
    depends_on:
      - postgres

  postgres:
    image: postgres:15-alpine
    container_name: mattermost-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: mmuser
      POSTGRES_PASSWORD: mmpassword
      POSTGRES_DB: mattermost
    volumes:
      - postgres-data:/var/lib/postgresql/data

volumes:
  mattermost-data:
  mattermost-logs:
  mattermost-config:
  postgres-data:
```

**`.env` file:**

```bash
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_DISCOVERY_URL=https://your-idp.com/.well-known/openid-configuration
```

**Deploy:**

```bash
docker-compose up -d
```

### Kubernetes

**Helm values (values-oidc.yaml):**

```yaml
# Using official Mattermost Helm chart with custom image
image:
  repository: your-registry/mattermost-oidc
  tag: "11.0"

config:
  MM_OPENIDSETTINGS_ENABLE: "true"
  MM_OPENIDSETTINGS_ID: "your-client-id"
  MM_OPENIDSETTINGS_DISCOVERYENDPOINT: "https://your-idp.com/.well-known/openid-configuration"
  MM_OPENIDSETTINGS_SCOPE: "openid email profile"
  MM_OPENIDSETTINGS_BUTTONTEXT: "Login with SSO"

extraEnvVars:
  - name: MM_OPENIDSETTINGS_SECRET
    valueFrom:
      secretKeyRef:
        name: mattermost-oidc-secret
        key: client-secret

# Ingress configuration
ingress:
  enabled: true
  hosts:
    - mattermost.example.com
  tls:
    - secretName: mattermost-tls
      hosts:
        - mattermost.example.com
```

**Create secret:**

```bash
kubectl create secret generic mattermost-oidc-secret \
  --from-literal=client-secret=your-client-secret
```

**Deploy with Helm:**

```bash
helm repo add mattermost https://helm.mattermost.com
helm install mattermost mattermost/mattermost \
  -f values-oidc.yaml \
  --namespace mattermost
```

### Terraform

If you're using Terraform with your infrastructure:

**`main.tf`:**

```hcl
# Example: AWS ECS deployment

resource "aws_ecs_task_definition" "mattermost" {
  family                   = "mattermost"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "2048"
  memory                   = "4096"
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name  = "mattermost"
      image = "your-registry/mattermost-oidc:11.0"

      portMappings = [
        {
          containerPort = 8065
          protocol      = "tcp"
        }
      ]

      environment = [
        {
          name  = "MM_OPENIDSETTINGS_ENABLE"
          value = "true"
        },
        {
          name  = "MM_OPENIDSETTINGS_ID"
          value = var.oidc_client_id
        },
        {
          name  = "MM_OPENIDSETTINGS_DISCOVERYENDPOINT"
          value = var.oidc_discovery_url
        },
        {
          name  = "MM_OPENIDSETTINGS_SCOPE"
          value = "openid email profile"
        },
        {
          name  = "MM_SERVICESETTINGS_SITEURL"
          value = "https://${var.domain_name}"
        }
      ]

      secrets = [
        {
          name      = "MM_OPENIDSETTINGS_SECRET"
          valueFrom = aws_secretsmanager_secret.oidc_secret.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.mattermost.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "mattermost"
        }
      }
    }
  ])
}

variable "oidc_client_id" {
  description = "OIDC Client ID"
  type        = string
}

variable "oidc_discovery_url" {
  description = "OIDC Discovery URL"
  type        = string
}
```

## Production Considerations

### SSL/TLS

Always use HTTPS in production:

1. Configure a reverse proxy (nginx, Traefik, Caddy) with TLS termination
2. Set `MM_SERVICESETTINGS_SITEURL` to your HTTPS URL
3. Ensure your IdP redirect URI uses HTTPS

**Example nginx config:**

```nginx
server {
    listen 443 ssl http2;
    server_name mattermost.example.com;

    ssl_certificate /etc/ssl/certs/mattermost.crt;
    ssl_certificate_key /etc/ssl/private/mattermost.key;

    location / {
        proxy_pass http://mattermost:8065;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### High Availability

For HA deployments:

1. Use a shared PostgreSQL database (or cluster)
2. Use shared file storage (S3, MinIO)
3. Deploy multiple Mattermost instances behind a load balancer
4. Use session affinity (sticky sessions) recommended but not required

### Monitoring

Monitor your deployment:

1. Enable Prometheus metrics:
   ```bash
   MM_METRICSSETTINGS_ENABLE=true
   MM_METRICSSETTINGS_LISTENADDRESS=":8067"
   ```

2. Configure health checks:
   - Health endpoint: `/api/v4/system/ping`
   - Expected response: `{"status":"OK"}`

### Backup Strategy

Back up regularly:

1. **Database**: Use pg_dump or your database's backup mechanism
2. **File storage**: Back up the data directory or S3 bucket
3. **Config**: Back up config.json or document environment variables

### Secrets Management

Never commit secrets. Use:

- Environment variables
- Kubernetes Secrets
- AWS Secrets Manager
- HashiCorp Vault
- Docker Secrets

## Upgrading

### Upgrading Mattermost

1. Review release notes for breaking changes
2. Back up database and files
3. Update the image tag in your deployment
4. Apply the update (rolling update if HA)
5. Verify OIDC login works after upgrade

### Upgrading the OIDC Module

1. Pull the latest mattermost-oidc code
2. Rebuild the custom image
3. Deploy the new image
4. Test OIDC authentication

## Troubleshooting

### Container won't start

Check logs:
```bash
docker logs mattermost
# or
kubectl logs deployment/mattermost
```

### OIDC redirects fail

1. Verify `MM_SERVICESETTINGS_SITEURL` matches your actual URL
2. Check the redirect URI in your IdP configuration
3. Ensure HTTPS is properly configured

### SSL certificate issues

If using self-signed certs for your IdP:
```bash
MM_SERVICESETTINGS_ENABLEINSECUREOUTGOINGCONNECTIONS=true  # Not recommended for production
```

Better: Add your CA certificate to the container's trust store.

### Connection refused to IdP

1. Check network connectivity from container to IdP
2. Verify DNS resolution
3. Check firewall rules

### Performance issues

1. Enable query logging to identify slow queries
2. Check database connection pool settings
3. Monitor memory and CPU usage
4. Consider horizontal scaling
