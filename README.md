<p align="center">
  <img src=".github/assets/logo.png" width="140" alt="Bulwark logo" />
</p>
<h1 align="center">Bulwark</h1>
<p align="center"><strong>Safe, policy-driven Docker container updater with digest-based change detection and rollback.</strong></p>
<p align="center">
  <a href="https://github.com/itsmrshow/bulwark/actions/workflows/docker-publish.yml">
    <img src="https://github.com/itsmrshow/bulwark/actions/workflows/docker-publish.yml/badge.svg?branch=main" alt="GitHub Build Status" />
  </a>
  <a href="https://hub.docker.com/r/itsmrshow/bulwark">
    <img src="https://img.shields.io/docker/v/itsmrshow/bulwark?sort=semver&label=Docker%20Hub%20Tag" alt="Docker Hub Tag" />
  </a>
  <a href="https://hub.docker.com/r/itsmrshow/bulwark">
    <img src="https://img.shields.io/docker/pulls/itsmrshow/bulwark" alt="Docker Hub Pulls" />
  </a>
  <a href="https://github.com/itsmrshow/bulwark">
    <img src="https://visitor-badge.laobi.icu/badge?page_id=itsmrshow.bulwark" alt="Visitors" />
  </a>
</p>

Bulwark manages Docker container updates without the chaos. It checks for new image digests, runs health probes after updating, and rolls back automatically if anything goes wrong.

- **Opt-in only** — containers need a `bulwark.enabled=true` label to be managed
- **Digest-based** — compares actual image digests, not tags
- **Policy tiers** — `notify` (alert only), `safe` (probes + rollback), `aggressive` (minimal checks)
- **Health gating** — HEALTHCHECK, HTTP, TCP, log, and stability probes
- **Auto-rollback** — reverts on probe failure, no manual intervention
- **Stateful-aware** — databases and other stateful services are blocked by default

## Quick Start

### Docker Compose (recommended)

```bash
cp .env.example .env
# Edit .env with your settings
docker compose pull bulwark
docker compose up -d bulwark
```

The Web Console is available at `http://localhost:8085` (default compose port).

### Updating Bulwark

Pull and recreate — don't rely on Bulwark to update itself:

```bash
docker compose pull bulwark
docker compose up -d bulwark
```

Bulwark skips self-updates during `apply` to avoid killing the process that's orchestrating updates. Set `BULWARK_ALLOW_SELF_UPDATE=true` to override this.

### Docker (single container)

```bash
docker run --rm -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /docker_data:/docker_data:ro \
  -e BULWARK_UI_ENABLED=true \
  -e BULWARK_UI_READONLY=true \
  itsmrshow/bulwark:latest
```

### Build from source

```bash
git clone https://github.com/itsmrshow/bulwark.git
cd bulwark
make build
```

Or install directly:

```bash
go install github.com/itsmrshow/bulwark/cmd/bulwark@latest
```

### CLI Usage

```bash
bulwark discover   # find managed targets
bulwark check      # check for updates
bulwark plan       # dry-run: see what would change
bulwark apply      # apply updates
bulwark serve      # start the web console
```

## Web Console

The UI is **read-only by default**. Write actions (apply, rollback) require a token.

```bash
# Enable writes
export BULWARK_UI_READONLY=false
export BULWARK_WEB_TOKEN="your-strong-token"
```

Add `Authorization: Bearer <token>` to write requests, or enter the token in the UI header.

### Local dev setup

```bash
# Terminal 1: API server
BULWARK_UI_ENABLED=true BULWARK_UI_READONLY=true bulwark serve

# Terminal 2: frontend dev server
cd web && npm install && npm run dev
```

Vite dev server runs on `http://localhost:5173` and proxies `/api` to `:8080`.

### Notifications

Discord and Slack webhooks can be configured in the Settings page. Supports immediate alerts on update discovery and scheduled digest summaries via cron.

Environment overrides (lock the values in the UI):

| Variable | Description |
|---|---|
| `DISCORD_WEBHOOK_URL` | Discord webhook URL |
| `SLACK_WEBHOOK_URL` | Slack webhook URL |
| `BULWARK_NOTIFY_ON_FIND` | Send alert immediately when updates found |
| `BULWARK_NOTIFY_DIGEST` | Enable digest summary |
| `BULWARK_NOTIFY_CHECK_CRON` | Override check schedule |
| `BULWARK_NOTIFY_DIGEST_CRON` | Override digest schedule |

Settings persist to `/data/bulwark.json` (configure with `BULWARK_DATA_DIR` or `BULWARK_CONFIG_PATH`).

## Labels

Everything is configured through container labels.

### Compose example

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

### Stateful service (protected)

```yaml
services:
  postgres:
    image: postgres:15
    labels:
      - bulwark.enabled=true
      - bulwark.policy=notify
      - bulwark.tier=stateful
```

### Loose container

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

### Label reference

**Core:**

| Label | Values | Default |
|---|---|---|
| `bulwark.enabled` | `true`/`false` | — (required) |
| `bulwark.policy` | `notify`, `safe`, `aggressive` | `safe` |
| `bulwark.tier` | `stateless`, `stateful` | `stateless` |
| `bulwark.definition` | `compose:/path/compose.yml#service=name` | — |

**Probes:**

| Label | Description |
|---|---|
| `bulwark.probe.type` | `http`, `tcp`, `log`, `stability` |
| `bulwark.probe.url` | HTTP probe URL |
| `bulwark.probe.expect_status` | Expected HTTP status (default: 200) |
| `bulwark.probe.tcp_host` | TCP probe host |
| `bulwark.probe.tcp_port` | TCP probe port |
| `bulwark.probe.log_pattern` | Regex pattern to match in logs |
| `bulwark.probe.stability_sec` | Stability window in seconds |

## Environment Variables

**Core:**

| Variable | Default | Description |
|---|---|---|
| `BULWARK_ROOT` | `/docker_data` | Base path for compose discovery |
| `BULWARK_STATE_DB` | `/var/lib/bulwark/state.db` | SQLite database path |
| `BULWARK_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

**Web Console:**

| Variable | Default | Description |
|---|---|---|
| `BULWARK_UI_ENABLED` | `true` | Enable the web console |
| `BULWARK_UI_READONLY` | `true` | Read-only mode |
| `BULWARK_WEB_TOKEN` | — | Bearer token for writes |
| `BULWARK_UI_ADDR` | `:8080` | Listen address |
| `BULWARK_UI_DIST` | `web/dist` | Built UI assets path |
| `BULWARK_PLAN_CACHE_TTL` | `15s` | Plan/overview cache TTL |
| `BULWARK_WEB_WRITE_RPS` | `1` | Write rate limit (req/s) |
| `BULWARK_WEB_WRITE_BURST` | `3` | Write burst capacity |

## Security

Bulwark requires `/var/run/docker.sock` access, which gives full Docker daemon control. Keep it on a trusted network.

- Web console is read-only by default
- Writes require bearer token auth
- Stateful services are protected from auto-updates
- Use a reverse proxy (Traefik, Caddy, etc.) for additional auth if exposing externally

## Development

```bash
make build          # build binary
make test           # run tests with race detection
make lint           # golangci-lint
make fmt            # format code
make docker-build   # build Docker image
```

## Roadmap

**Done:**
- Discovery engine (compose projects + standalone containers)
- Docker Hub digest checking
- Policy engine (notify/safe/aggressive)
- SQLite state persistence
- Update executor with compose awareness
- Health probes (HTTP, TCP, Docker HEALTHCHECK, stability)
- Automatic rollback on probe failure
- Cron-based scheduler
- Web console with React frontend
- REST API with async operation tracking
- Notification system (Discord/Slack)

**Planned:**
- Webhook receiver for Docker Hub / Harbor push events
- Prometheus metrics endpoint
- Grafana dashboard templates
- Log pattern probe (regex-based health checks)

**Future ideas:**
- Private registry auth and broader registry support
- Email / PagerDuty notifications
- Update windows (time-based restrictions)
- Canary deployments
- Pre/post update hooks

## License

MIT — see [LICENSE](LICENSE) for details.

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
