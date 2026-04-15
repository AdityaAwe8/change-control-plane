# Contributing

## Principles

ChangeControlPlane is being built as a serious commercial-grade control plane. Contributions should optimize for:

- clear domain boundaries
- explainable behavior
- secure defaults
- predictable APIs
- auditability
- maintainability over cleverness

## Development Workflow

1. Read the relevant architecture docs and ADRs before large changes.
2. Keep module boundaries explicit. Prefer adding a new package seam over leaking concerns across domains.
3. Add or update tests for behavior changes.
4. Update docs and OpenAPI when API contracts change.
5. Preserve deterministic core behavior. AI or heuristic layers should remain optional and additive.

## Coding Standards

- Go: favor small packages, explicit types, and standard library primitives where practical.
- TypeScript: keep UI structure intentional and production-minded. Avoid throwaway dashboards.
- Python: reserve for analytics, simulation, and advanced scoring where its ecosystem materially helps.
- SQL: use PostgreSQL idioms with multi-tenant boundaries and audit-friendly metadata.

## Testing

Run locally before proposing changes:

```bash
make fmt
make test
make build
```

For frontend work:

```bash
make web-install
make web-build
```

## Documentation

Any significant architectural decision should be captured in:

- `docs/architecture/*` for system design
- `docs/adr/*` for irreversible or high-impact choices
- `docs/api/openapi.yaml` for API contract changes

## Security

- never commit secrets
- avoid logging sensitive configuration
- thread organization and project ownership through new domain models
- keep authorization and policy hooks near critical boundaries
