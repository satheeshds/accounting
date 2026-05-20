# Copilot instructions

## Build and test

- Build the API binary with `make build` or `go build .`.
- Run the full Go test suite with `make test` or `go test ./...`.
- Run a single test with `go test ./models -run '^TestNormalizeDate$'` (swap the package path and test name as needed).
- Start the full local stack with `make compose-up` and stop it with `make compose-down`.
- Regenerate embedded Swagger docs with `make swag-init` after changing `main.go`, `handlers/`, `models/`, or `store/`.
- When working on Windows, `go test ./...` can fail in `github.com/duckdb/duckdb-go` bindings even when `go build .` succeeds. If that happens, retry the full suite in Linux or the containerized stack before assuming the app code is broken.

## High-level architecture

- This repo has two executables:
  - `main.go` runs the HTTP API, serves the embedded `static/` UI, and serves the embedded Swagger JSON/UI.
  - `platform/main.go` is the background worker that migrates every tenant at startup, backfills recurring-payment occurrences, then regenerates occurrences once per day.
- `docker-compose.yml` reflects the deployed shape: `portal` API, `platform` worker, `nexus-gateway`, and `nexus-control`.
- Runtime config is loaded once at startup with `handlers.Configure(handlers.ConfigFromEnv())`; handlers then read package-level config rather than re-reading env vars.
- Authentication and database access are tied together in Nexus mode:
  - `/api/v1/auth/register` and `/api/v1/auth/login` proxy to `nexus-control`.
  - `handlers.BearerAuth` accepts JWT bearer tokens or Nexus service-account basic auth.
  - For authenticated requests, middleware opens a per-request tenant DB connection with `db.OpenWithCredentials(...)` and stores it in request context.
  - `main.go` does not open a global production DB connection up front; normal request flow depends on middleware injecting one.
- The main request path is `handler -> store.New(getDB(r)) -> store methods -> db.PortalDB`.
- `transaction_documents` is the central allocation/link table across bills, invoices, payouts, transactions, and recurring payment occurrences. Matching, allocated/unallocated amounts, and status updates all build on that table.
- Recurring-payment matching works against generated `recurring_payment_occurrences`, not directly against the parent `recurring_payments` row. The platform service is responsible for creating catch-up occurrences from `next_due_date`.

## Key conventions

- Prefer semantic code search with `semble search` and `semble find-related` before falling back to literal grep. The repo already documents this in `AGENTS.md` and `.github/agents/semble-search.md`.
- The app serves embedded generated Swagger output from `docs/`; after API shape changes, regenerate `docs/swagger.json` and `docs/swagger.yaml` with `make swag-init` instead of treating `openapi.yaml` as the runtime source of truth.
- Application SQL should usually use bare table names plus `?` placeholders. `db.PortalDB` rewrites those queries to `lake.<table>` and `$N` placeholders automatically.
- Migrations are the exception to the SQL rewrite rule: migration files in `db/migrations/` run through Goose on the raw `*sql.DB`, so write migration SQL explicitly for that environment.
- Keep request validation in `models.*Input.Validate()`. Handlers usually just decode JSON, call `Validate()`, delegate to the store layer, and respond through the shared `Response` envelope via `writeJSON` / `writeError`.
- Auth handlers are special-cased: successful login responses are re-wrapped in the standard envelope, but proxied Nexus error responses may be passed through as-is.
- Tests often rely on the package-level `handlers.DB` fallback, but production code should prefer request-scoped DB access through middleware context.
- Bills and invoices scan date fields through `store.nullableDate` so values coming back as `time.Time`, plain dates, or timestamp strings all normalize to `YYYY-MM-DD`.
- Account balances, document allocation totals, unallocated amounts, and recurring-occurrence payment status are derived in store queries/helpers. Do not duplicate those calculations in handlers.
