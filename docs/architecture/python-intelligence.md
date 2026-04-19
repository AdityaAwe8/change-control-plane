# Python Intelligence

## Purpose

The Python subsystem provides supplemental analytics and simulation for ChangeControlPlane without taking ownership of the authoritative online control path.

The design choice is deliberate:

- Go remains responsible for the primary deterministic API, persistence, auth, RBAC, audit, and rollout state transitions.
- Python handles model-adjacent logic that benefits from a flexible analytics runtime.
- The product can grow into richer statistical or ML-assisted analysis without moving core governance out of the main application.

## Current Runtime Boundary

The Go application invokes [python/intelligence_cli.py](/Users/aditya/Documents/ChangeControlPlane/python/intelligence_cli.py) as a subprocess.

Contract:

- input: JSON over `stdin`
- output: JSON over `stdout`
- commands:
  - `risk-augment`
  - `rollout-simulate`

Go-side adapter:

- [internal/intelligence/python.go](/Users/aditya/Documents/ChangeControlPlane/internal/intelligence/python.go)

Python implementation:

- [python/risk_models/explainable.py](/Users/aditya/Documents/ChangeControlPlane/python/risk_models/explainable.py)
- [python/analytics/history.py](/Users/aditya/Documents/ChangeControlPlane/python/analytics/history.py)
- [python/simulation/rollout.py](/Users/aditya/Documents/ChangeControlPlane/python/simulation/rollout.py)

## What Is Implemented

### Risk Augmentation

Python currently computes:

- normalized change-surface factors
- change clustering
- historical-pattern summaries
- confidence adjustments
- supplemental explanations
- extra guardrail recommendations

These results are merged into:

- `risk_assessments.explanation`
- `risk_assessments.recommended_guardrails`
- `risk_assessments.metadata["python_intelligence"]`

### Rollout Simulation

Python currently computes:

- recommended next action
- rollout hotspots
- timeline notes
- verification-focus signals
- simulation metadata such as failure modes and observation-window guidance

These results are merged into:

- `rollout_plans.verification_signals`
- `rollout_plans.explanation`
- `rollout_plans.metadata["python_simulation"]`

## Failure Model

Python is supplemental, not authoritative.

If the Python subprocess is unavailable or returns an error:

- the deterministic Go path still succeeds
- the persisted record notes that supplemental intelligence was unavailable
- the control plane retains explainable baseline behavior

This preserves product reliability while still making Python a real runtime dependency when available.

## Verification

The Python subsystem is currently verified by:

- Python unit and CLI contract tests in [python/tests/test_intelligence.py](/Users/aditya/Documents/ChangeControlPlane/python/tests/test_intelligence.py)
- Go adapter tests in [internal/intelligence/python_test.go](/Users/aditya/Documents/ChangeControlPlane/internal/intelligence/python_test.go)
- end-to-end API persistence coverage in [internal/app/http_intelligence_test.go](/Users/aditya/Documents/ChangeControlPlane/internal/app/http_intelligence_test.go)

## Current Limits

The Python subsystem is real, but still intentionally limited:

- no separate Python service process
- no model registry
- no learned model training pipeline
- no historical warehouse or offline analytics jobs
- no stochastic simulation or traffic replay
- no live signal ingestion from external providers

That limitation is intentional. The current design establishes a production-leaning boundary first and leaves room for deeper model sophistication later.
