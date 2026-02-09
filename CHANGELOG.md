# Changelog

All notable changes to Bulwark will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-29

### Added - Core Functionality

- **Discovery Engine** - Automatic discovery of Docker Compose projects and labeled containers
  - Scans `/docker_data` for compose files
  - Discovers loose containers with `bulwark.enabled=true` label
  - Auto-detects known databases (postgres, mysql, redis, etc.) as stateful
  - Supports both map and array label formats in docker-compose.yml

- **Registry Integration** - Docker Hub digest checking
  - Fetches remote image digests via Docker Registry v2 API
  - Multi-platform manifest support (linux/amd64 preferred)
  - Digest comparison for reliable update detection
  - Docker Hub authentication support

- **Policy Engine** - Three-tier update policy system
  - `notify` - Check only, never auto-update (databases default)
  - `safe` - Update with full health probes (recommended)
  - `aggressive` - Update with minimal checks (use carefully)
  - Tier-based protection (stateless vs stateful services)

- **SQLite State Persistence** - Comprehensive state tracking
  - Targets, services, and update history tables
  - Automatic schema initialization and migrations
  - WAL mode for better concurrency
  - Proper foreign key constraints

- **Update Executor** - Safe container updates
  - Compose project updates via `docker compose pull && up`
  - Loose container updates via definition labels
  - Per-target locking (prevents concurrent updates)
  - Digest pinning for reliable rollback

- **Health Probe System** - Multi-tier health verification
  - HTTP probes with custom status codes
  - TCP connection probes
  - Docker HEALTHCHECK status verification
  - Stability window probes (wait N seconds)
  - Configurable retries and timeouts

- **Automatic Rollback** - Zero-downtime failure recovery
  - Automatic rollback on probe failure
  - Pull previous digest + tag + recreate
  - Rollback status tracked in update history
  - Manual rollback via API endpoint

- **Scheduler** - Cron-based automation
  - Flexible cron expression support
  - Separate check and apply jobs
  - 30-minute timeout per job
  - Graceful shutdown support

### Added - Web Console

- **Production-Ready UI** - Full-featured web interface
  - React + TypeScript + Vite
  - Tailwind CSS with shadcn-inspired components
  - Overview, Targets, Plan, Apply, History pages
  - Read-only mode by default
  - Token-based authentication for write operations
  - Responsive design

- **REST API Backend** - Comprehensive API
  - `GET /api/health` - Service health check
  - `GET /api/overview` - Dashboard summary
  - `GET /api/targets` - List managed targets
  - `GET /api/plan` - Structured update plan
  - `POST /api/apply` - Async update execution
  - `GET /api/runs/{id}` - Real-time run status
  - `POST /api/rollback` - Manual rollback
  - `GET /api/history` - Update history
  - Bearer token auth middleware
  - Rate limiting on write endpoints
  - CORS and security headers

- **Async Run Manager** - Background job execution
  - In-memory run tracking
  - Real-time event streaming
  - Progress updates during apply
  - Automatic cleanup of old runs

### Added - CLI Commands

- `bulwark discover` - List all managed targets
- `bulwark check` - Check for available updates
- `bulwark plan` - Dry-run with structured output
- `bulwark apply` - Apply updates with probes and rollback
- `bulwark serve` - Run web console + API server

### Added - Docker & Deployment

- Multi-stage Dockerfile (Node → Go → Runtime)
- Docker Compose file for production deployment
- GitHub Actions CI/CD workflow
- Volume mounts for Docker socket and data
- Environment-based configuration

### Added - Documentation

- Comprehensive README with examples
- Label reference guide
- Environment variable documentation
- Security considerations
- Web console setup instructions
- Docker deployment guide

### Technical Details

- **Language**: Go 1.24
- **Dependencies**:
  - `github.com/docker/docker` v25.0.5 - Docker API
  - `github.com/mattn/go-sqlite3` v1.14.33 - SQLite driver
  - `github.com/robfig/cron/v3` v3.0.1 - Cron scheduler
  - `github.com/rs/zerolog` v1.34.0 - Structured logging
  - `github.com/spf13/cobra` v1.10.2 - CLI framework

- **Testing**: Unit tests for API, executor, planner, auth middleware
- **Logging**: Structured JSON logs with secret redaction
- **Security**: Read-only by default, token-based auth, rate limiting

### Known Limitations

- Docker Hub only (no GitLab/GitHub/Harbor support yet)
- No webhook server (registry push notifications)
- No Prometheus metrics endpoint
- No log pattern probe implementation
- No email/Slack notifications

## Future Releases

See README.md roadmap for planned v1.1 and v2.0 features.

---

[1.0.0]: https://github.com/itsmrshow/bulwark/releases/tag/v1.0.0
