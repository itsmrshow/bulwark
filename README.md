# Bulwark

**Safe, policy-driven Docker container updater with digest-based change detection and rollback capability.**

Bulwark is a Docker container update management tool designed for safety, transparency, and control. Unlike aggressive auto-updaters, Bulwark provides:

- **Opt-in only** - Nothing updates without explicit configuration
- **Digest-based detection** - Reliable change detection via image digests
- **Policy tiers** - notify (check only), safe (full probes), aggressive (minimal checks)
- **Health gating** - Multi-tier probes: HEALTHCHECK → HTTP → TCP → Log → Stability
- **Automatic rollback** - Reverts to previous digest on probe failure
- **Stateful protection** - Never auto-updates databases without explicit override

## Features

- **Discovery**: Scans Docker Compose projects and labeled containers
- **Update Detection**: Compares local vs remote registry digests
- **Safe Updates**: Applies updates with health probes and rollback
- **Flexible Triggers**: Scheduled (cron) or webhook-triggered
- **Compose-aware**: Uses `docker compose` for proper project handling
- **Loose Container Support**: Manages standalone containers via definition labels

## Quick Start

### Installation

```bash
go install github.com/yourusername/bulwark/cmd/bulwark@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/bulwark.git
cd bulwark
make build
```

### Basic Usage

```bash
# Discover managed targets
bulwark discover

# Check for available updates
bulwark check

# Plan updates (dry-run)
bulwark plan

# Apply updates
bulwark apply

# Run the Web Console (API + UI)
bulwark serve
```

## Web Console (UI)

Bulwark includes a production-ready Web Console that is **read-only by default**. Write actions (apply/rollback) must be explicitly enabled and are protected by a bearer token.

### Local development

```bash
# Terminal 1: run the API/UI server
export BULWARK_UI_ENABLED=true
export BULWARK_UI_READONLY=true
bulwark serve

# Terminal 2: run the frontend dev server
cd web
npm install
npm run dev
```

Open `http://localhost:5173` for the Vite dev server (it proxies `/api` to `http://localhost:8080`).

### Docker

```bash
docker build -t bulwark:dev .
docker run --rm -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /docker_data:/docker_data \
  -e BULWARK_UI_ENABLED=true \
  -e BULWARK_UI_READONLY=true \
  bulwark:dev
```

Visit `http://localhost:8080` to access the Web Console.

### Enabling write actions

```bash
export BULWARK_UI_READONLY=false
export BULWARK_WEB_TOKEN="your-strong-token"
```

Then add `Authorization: Bearer <token>` for write requests, or enter the token in the UI header.

## Label Configuration

Enable Bulwark management on your containers using labels:

### Compose Project

```yaml
services:
  nginx:
    image: nginx:latest
    labels:
      - bulwark.enabled=true
      - bulwark.policy=safe
      - bulwark.tier=stateless
      - bulwark.probe.type=http
      - bulwark.probe.url=http://localhost:80
      - bulwark.probe.expect_status=200
```

### Stateful Service (Protected)

```yaml
services:
  postgres:
    image: postgres:15
    labels:
      - bulwark.enabled=true
      - bulwark.policy=notify  # Only notify, never auto-update
      - bulwark.tier=stateful
```

### Loose Container

```bash
docker run -d \
  --name myapp \
  --label bulwark.enabled=true \
  --label bulwark.policy=safe \
  --label bulwark.definition=compose:/docker_data/myapp/docker-compose.yml#service=myapp \
  --label bulwark.probe.type=tcp \
  --label bulwark.probe.tcp_host=localhost \
  --label bulwark.probe.tcp_port=8080 \
  myapp:latest
```

## Label Reference

### Core Labels

- `bulwark.enabled=true|false` - Enable Bulwark management (required)
- `bulwark.policy=notify|safe|aggressive` - Update policy (default: safe)
- `bulwark.tier=stateless|stateful` - Service tier (default: stateless)
- `bulwark.definition=compose:/abs/path/compose.yml#service=<service>` - For loose containers

### Probe Labels

- `bulwark.probe.type=http|tcp|log|stability` - Probe type
- `bulwark.probe.url=<url>` - HTTP probe URL
- `bulwark.probe.expect_status=<code>` - Expected HTTP status (default: 200)
- `bulwark.probe.tcp_host=<host>` - TCP probe host
- `bulwark.probe.tcp_port=<port>` - TCP probe port
- `bulwark.probe.log_pattern=<regex>` - Log pattern to match
- `bulwark.probe.stability_sec=<seconds>` - Stability window duration

## Environment Variables

- `BULWARK_ROOT=/docker_data` - Base path for compose discovery
- `BULWARK_STATE_DB=/var/lib/bulwark/state.db` - State database location (SQLite)
- `BULWARK_LOG_LEVEL=debug|info|warn|error` - Logging verbosity

### Web Console

- `BULWARK_UI_ENABLED=true|false` - Enable the Web Console (default: true)
- `BULWARK_UI_READONLY=true|false` - Read-only mode (default: true)
- `BULWARK_WEB_TOKEN=...` - Required bearer token for write actions
- `BULWARK_UI_ADDR=:8080` - UI/API listen address
- `BULWARK_UI_DIST=web/dist` - Path to built UI assets
- `BULWARK_PLAN_CACHE_TTL=15s` - Cache TTL for overview/plan calculations
- `BULWARK_WEB_WRITE_RPS=1` - Write endpoint rate limit (requests per second)
- `BULWARK_WEB_WRITE_BURST=3` - Write endpoint burst capacity

## Configuration File

Create `/etc/bulwark/config.yaml`:

```yaml
discovery:
  base_path: "/docker_data"
  scan_interval: "5m"

scheduler:
  enabled: true
  check_cron: "0 */6 * * *"  # Every 6 hours
  apply_cron: "0 2 * * *"    # 2 AM daily

webhook:
  enabled: true
  listen_addr: ":8080"
  token: "${BULWARK_WEBHOOK_TOKEN}"
  allowed_ips:
    - "10.0.0.0/8"

state:
  backend: "sqlite"
  path: "/var/lib/bulwark/state.db"

logging:
  level: "info"
  format: "json"
```

## Security Considerations

⚠️ **Docker Socket Access**: Bulwark requires access to `/var/run/docker.sock`, which provides full Docker daemon control. Run Bulwark in a trusted environment only.

- Web Console is read-only by default
- Write actions require `BULWARK_WEB_TOKEN` and bearer auth
- Keep the API on an internal network and use a reverse proxy (Traefik/HAProxy/Cloudflare Access) for additional auth
- Optional IP allowlist for additional security
- Secrets are redacted from logs automatically
- Stateful services (databases) protected by default

## Development

```bash
# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint

# Build Docker image
make docker-build
```

## Roadmap

### v1.0 (Complete! 🎉)

- [x] Basic project structure
- [x] Docker client integration (Phase 1)
- [x] CLI framework (Phase 1)
- [x] Discovery engine (Phase 2) - Compose + loose containers
- [x] Registry digest checking (Phase 3) - Docker Hub
- [x] Policy engine (Phase 3) - notify/safe/aggressive
- [x] SQLite state persistence (Phase 5)
- [x] Update executor (Phase 6) - Compose + container updates
- [x] Health probes (Phase 7) - HTTP/TCP/Docker/Stability
- [x] Automatic rollback (Phase 8) - On probe failure
- [x] Scheduler (Phase 9) - Cron-based automation
- [x] Web UI dashboard (Phases 1-7) - Production-ready React app
- [x] API backend - REST API with async operations

### v1.1 (Planned)

- [ ] Webhook server - Docker Hub/Harbor push notifications
- [ ] Prometheus metrics endpoint
- [ ] Grafana dashboard templates
- [ ] Log pattern probe (regex-based health checks)

### v2.0 (Future)

- Multi-registry support (GitLab, GitHub, Harbor, GHCR)
- Notification system (Slack, Discord, email, PagerDuty)
- Update windows (time-based restrictions)
- Canary deployments (gradual rollouts)
- Pre/post update hooks (custom scripts)

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions welcome! Please see CONTRIBUTING.md for guidelines.
