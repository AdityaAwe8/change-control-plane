# Full Verification Plan

This document tracks the current full-system verification strategy for Change Control Plane. It is intentionally evidence-based. A subsystem is not considered validated because it exists in the tree, because it compiles, or because it has a happy-path test somewhere in the repo.

## Verification Principles

- Prefer executable proof over narrative confidence.
- Validate happy path and at least one important failure path.
- Validate RBAC and tenant boundaries for every sensitive path.
- Validate persistence or side effects for every mutating path.
- Treat the web UI as a product surface, not a static shell.
- Treat simulated provider paths and near-real provider paths differently in documentation.

## Verification Layers

1. Unit tests
   - deterministic engines
   - transition legality
   - provider normalization
   - signal normalization
   - Python intelligence logic
2. Integration tests
   - HTTP handlers through app layer
   - auth, RBAC, and tenant boundaries
   - PostgreSQL repository behavior
   - rollout, verification, rollback, and status-history persistence
3. Browser interaction tests
   - login/session
   - primary workflow forms and buttons
   - permission-aware visibility
   - rollout and status dashboard surfaces
4. Smoke verification
   - persisted API flow through auth -> creation -> change -> risk -> rollout -> execution -> verification -> rollback -> status history -> audit
5. Documentation truth pass
   - downgrade claims not backed by proof
   - mark simulated vs near-real vs blocked explicitly

## Current High-Risk Verification Targets

1. Browser usability across cross-origin dev ports
   - fixed in this pass by adding API CORS handling and browser tests against the real API
2. Status-history search and rollback visibility
   - verified through HTTP, PostgreSQL, browser, and smoke paths
3. CLI query correctness
   - hardened in this pass by URL-encoding status filters and adding command tests
4. Near-real provider seams
   - still limited to unit-level proof; not promoted to fully live external integration
5. Thin read-only surfaces
   - several web routes still render meaningful content but have no route-specific interaction tests

## Required Local Verification Commands

```bash
go vet ./...
go test ./...
python3 -m unittest discover -s python/tests -v
cd /Users/aditya/Documents/ChangeControlPlane/web && pnpm typecheck
cd /Users/aditya/Documents/ChangeControlPlane/web && pnpm build
cd /Users/aditya/Documents/ChangeControlPlane/web && pnpm test:e2e
BASE_URL=http://127.0.0.1:18084 ./scripts/smoke.sh
```

## Required CI Verification

- Go format check
- `go vet ./...`
- `go test ./...`
- PostgreSQL-backed repository and integration tests
- Python unit tests
- web typecheck and build
- browser interaction tests
- persisted smoke flow

## Review Discipline

When a gap remains:

- mark it partial or blocked in the matrix
- record it in the residual risk register
- avoid upgrading docs or README language beyond the proof level
