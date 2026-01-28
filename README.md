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

# Run as daemon with webhook + scheduler
bulwark serve
```

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
- `BULWARK_STATE_DIR=/data/state` - State database location
- `BULWARK_MODE=schedule|webhook|both` - Operating mode
- `BULWARK_INTERVAL=15m` - Scheduler interval
- `BULWARK_WEBHOOK_ENABLED=true|false` - Enable webhook server
- `BULWARK_WEBHOOK_ADDR=:8080` - Webhook listen address
- `BULWARK_WEBHOOK_TOKEN=...` - Required bearer token for webhooks
- `BULWARK_DRY_RUN=true|false` - Global dry-run mode
- `BULWARK_LOG_LEVEL=debug|info|warn|error` - Logging verbosity

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

- Webhook server is disabled by default
- Webhook requires bearer token authentication
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

### v1.0 (Current)
- [x] Basic project structure
- [x] Docker client integration
- [x] CLI framework
- [ ] Discovery engine
- [ ] Registry digest checking
- [ ] Policy engine
- [ ] Update executor
- [ ] Health probes
- [ ] Rollback manager
- [ ] Webhook server
- [ ] Scheduler

### v2.0 (Future)
- Multi-registry support (GitLab, GitHub, Harbor)
- Notification system (Slack, Discord, email)
- Web UI dashboard
- Prometheus metrics
- Update windows (time-based restrictions)
- Canary deployments
- Pre/post hooks

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions welcome! Please see CONTRIBUTING.md for guidelines.
