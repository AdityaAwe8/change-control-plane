# Advisory Mode

Advisory mode is the safest brownfield adoption path for the current platform.

## What Advisory Mode Does

- ingests real change metadata
- observes backend/runtime state
- collects runtime signals
- records verification evidence
- stores recommendations such as advisory rollback or advisory pause

## What Advisory Mode Does Not Do

- it does not execute live backend control actions during reconcile
- it does not submit, pause, resume, or rollback the external deployment target for non-simulated integrations

## Current Safety Mechanism

For live-style backend integrations:

- if the integration is not in `active_control`
- or `control_enabled` is false

the runtime path downgrades provider actions into observation-only sync behavior.

## Current Gaps

- broader UI messaging still needs to distinguish recommendation from execution more clearly
- manual operator action hardening is still incomplete
- policy expressions still evaluate into deterministic decisions rather than richer advisory narratives
