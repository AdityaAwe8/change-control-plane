# Change Control Plane Python Intelligence

This workspace contains the deterministic Python intelligence layer that extends the Go control-plane baseline.

It is intentionally scoped to supplemental analytics and simulation work:

- explainable risk augmentation
- normalized factor calculation
- historical-pattern analysis groundwork
- rollout simulation groundwork
- structured JSON interfaces for the Go API and worker runtimes

The Go runtime remains the authoritative online decision engine. Python is used for supplemental analytics so the platform can evolve toward richer model-driven behavior without moving core authorization, persistence, or deterministic policy paths out of Go.

## Runtime Boundary

The Go application invokes [intelligence_cli.py](/Users/aditya/Documents/ChangeControlPlane/python/intelligence_cli.py) as a subprocess and exchanges JSON over `stdin` and `stdout`.

Supported commands:

- `risk-augment`
- `rollout-simulate`

## Local Verification

```bash
python3 -m unittest discover -s python/tests -v
python3 python/intelligence_cli.py risk-augment < python/tests/fixtures/risk_payload.json
```

## Layout

```text
python/
  analytics/
  common/
  risk_models/
  simulation/
  tests/
  intelligence_cli.py
  pyproject.toml
```
