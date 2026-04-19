# Webhook Registration Status

## Classification

| Provider | Status | Reality |
| --- | --- | --- |
| GitHub | partial | Automatic organization-scoped webhook registration and repair are real when the integration has usable owner scope, token/app auth, and a configured webhook secret env reference. |
| GitLab | partial | Automatic group-scoped webhook registration and repair are real when the integration has group scope, token auth, and a configured webhook secret env reference. |
| Delivery health tracking | real and credible | Registration status, delivery health, last registration, last delivery, failure count, and latest error are persisted and visible. |
| Automatic secret generation | missing | Operators still provide webhook secret references; the platform does not yet generate and escrow those secrets for them. |
| Automatic teardown | partial | Registration repair/update exists, but full deletion/teardown lifecycle is not yet a polished operator flow. |

## Honest Limits

- Automatic registration only works for supported GitHub/GitLab scopes.
- Missing scope metadata or secret references fall back to `manual_required` or `error`, not silent success.
- Delivery health is integration-scoped and useful, but still not a full provider-native webhook observability model.
