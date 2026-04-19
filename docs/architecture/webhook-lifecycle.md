# Webhook Lifecycle

Webhook registration is now more complete than the original happy-path auto-registration milestone, but it is still intentionally honest about its limits.

## What Exists

- automatic registration and repair sync for supported GitHub and GitLab paths
- persisted webhook-registration records per integration instance
- delivery health tracking
- registration status reconciliation for:
  - `not_registered`
  - `registered`
  - `repair_recommended`
  - `manual_required`
  - `disabled`
- operator-visible last registration, validation, delivery, and latest error details

## What Remains Partial

- secret generation and rotation are still env-ref driven, not product-managed
- teardown and delete lifecycle is still limited
- broader hosted-provider proof is still missing
