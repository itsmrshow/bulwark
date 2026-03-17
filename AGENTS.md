# Repository Guidelines

## Project Structure & Module Organization
`cmd/bulwark` contains the CLI entrypoint. Core backend packages live under `internal/` and are grouped by concern, such as `api/`, `cli/`, `discovery/`, `executor/`, `planner/`, `probe/`, `scheduler/`, and `state/`. Integration fixtures and higher-level test assets live in `tests/`. The web console lives in `web/`: application code is in `web/src`, static assets in `web/public`, and the production build output in `web/dist`.

## Build, Test, and Development Commands
Use `make build` to compile the Go binary into `build/bulwark`. Use `make test` to run backend tests with race detection and coverage output (`coverage.out`). Use `make fmt` for Go formatting and `make lint` for `golangci-lint`.

For frontend work, run `cd web && npm install` once, then:

- `npm run dev` starts the Vite dev server on `:5173`
- `npm run build` creates the production bundle
- `npm test` runs Vitest once
- `npm run test:watch` runs frontend tests in watch mode

For local full-stack development, run `bulwark serve` with UI env vars from the README, then start the frontend dev server in `web/`.

## Coding Style & Naming Conventions
Follow standard Go formatting with tabs and `gofmt` via `make fmt`. Keep Go packages lowercase and focused; export only cross-package APIs. Frontend code uses TypeScript, React function components, and PascalCase filenames for components and pages, such as `SettingsPage.tsx` or `RiskBadge.tsx`. Prefer colocated helpers under `web/src/lib` and put UI primitives under `web/src/components/ui`.

## Testing Guidelines
Backend tests use Go’s `testing` package and follow the `*_test.go` convention next to the code they cover. Frontend tests use Vitest with Testing Library in `web/src/components/__tests__` and `*.test.tsx` naming. Add tests with every behavior change, especially around update planning, probes, API handlers, and scheduler logic. Run `make test` and `cd web && npm test` before opening a PR.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commit prefixes like `feat:`, `fix:`, `docs:`, and `test:`. Keep subjects short and imperative, for example `feat: add safe auto-update gating`. PRs should include a concise description, linked issue when applicable, test coverage notes, and screenshots for UI changes. Do not commit `build/`, `web/dist`, or other generated artifacts unless the change explicitly requires them.
