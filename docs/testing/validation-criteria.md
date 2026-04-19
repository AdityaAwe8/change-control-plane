# Validation Criteria

This document defines what Change Control Plane means by `validated`.

## General Rule

A flow is only validated if:

- the happy path is proven
- at least one important failure path is proven
- auth or permission behavior is proven where relevant
- persistence or side effects are proven where relevant
- response or output shape is proven where relevant
- audit or status-event side effects are proven where relevant
- documentation does not overclaim beyond the proof

## UI Validation

A UI action is only validated if:

- the control exists in the rendered browser UI
- it triggers the correct request or action
- success behavior is proven
- failure behavior is proven
- permission visibility is proven where relevant
- resulting backend state change is proven if applicable

The UI is only `validated and hardened` when the control also has sane busy or feedback behavior rather than silent failure.

## API Validation

An API endpoint is only validated if:

- request validation is exercised
- success response is exercised
- at least one failure response is exercised
- RBAC or tenant behavior is exercised where relevant
- database or side-effect behavior is exercised if mutating
- audit or status-event side effects are exercised if applicable

## Database Validation

A DB path is only validated if:

- written data is correct
- retrieved data is correct
- filtering or scoping is correct
- not-found or duplicate behavior is correct
- transaction semantics are correct if the path is multi-step

## Control-Loop Validation

A control-loop path is only validated if:

- transition legality is proven
- the worker or reconcile path is exercised
- duplicate action prevention is exercised where relevant
- signal or provider inputs are exercised
- persisted execution, verification, audit, and status results are exercised

## Provider Validation

- `simulated and verified` means the provider path is executable end to end in tests and smoke flows
- `near-real and verified` means a real client boundary exists and is tested against realistic responses, but the repo has not proven live external environment execution
- `live and verified` means the repo has proven the path against a real external dependency
