# Persistence Architecture

ChangeControlPlane now supports a PostgreSQL-backed runtime behind the existing application store seam.

## Runtime Shape

- `internal/storage/contracts.go` defines the storage interface used by the app layer
- `internal/app/store.go` remains as the in-memory implementation for fast tests and fallback use
- `internal/storage/postgres.go` provides the persisted implementation
- `internal/storage/migrations.go` applies ordered SQL migrations

## Design Notes

- the app layer depends on interfaces, not SQL
- the PostgreSQL store owns SQL, JSON marshaling, and transaction handling
- `WithinTransaction` is part of the store seam so multi-step flows can stay in the app layer without leaking SQL
- tenant scoping is expressed as organization- and project-aware query types

## Operational Model

- local and default runtime storage is PostgreSQL via `CCP_STORAGE_DRIVER=postgres`
- tests can continue to use the in-memory store where determinism and speed matter more than integration depth
- migrations are applied by `cmd/migrate` and can also run automatically when `CCP_AUTO_MIGRATE=true`
