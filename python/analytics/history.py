from __future__ import annotations

from common.models import average, bool_factor, bounded_ratio


def classify_change_cluster(change: dict, service: dict, environment: dict) -> str:
    if change.get("touches_iam") or change.get("touches_secrets"):
        return "identity-and-access"
    if change.get("touches_schema"):
        return "schema-sensitive"
    if environment.get("production") and service.get("customer_facing"):
        return "customer-facing-production"
    if change.get("touches_infrastructure"):
        return "infrastructure-coupled"
    if change.get("dependency_changes"):
        return "dependency-refresh"
    return "routine-application"


def analyze_change_pattern(payload: dict) -> dict:
    change = payload["change"]
    service = payload["service"]
    environment = payload["environment"]
    cluster = classify_change_cluster(change, service, environment)

    volatility_index = average(
        [
            bounded_ratio(float(change.get("file_count", 0)), 50.0),
            bounded_ratio(float(change.get("resource_count", 0)), 10.0),
            bounded_ratio(float(change.get("historical_incident_count", 0)), 5.0),
            bool_factor(bool(change.get("touches_infrastructure"))),
            bool_factor(bool(change.get("touches_schema"))),
            bool_factor(bool(change.get("poor_rollback_history"))),
        ]
    )

    observability_coverage = 1.0 - average(
        [
            0.0 if service.get("has_observability") else 1.0,
            0.0 if service.get("has_slo") else 1.0,
        ]
    )

    customer_exposure = average(
        [
            bool_factor(bool(environment.get("production"))),
            bool_factor(bool(service.get("customer_facing"))),
            bounded_ratio(float(service.get("dependent_services_count", 0)), 4.0),
        ]
    )

    return {
        "cluster": cluster,
        "volatility_index": round(volatility_index, 3),
        "historical_incident_bias": round(
            bounded_ratio(float(change.get("historical_incident_count", 0)), 5.0), 3
        ),
        "rollback_fragility": round(
            average(
                [
                    bool_factor(bool(change.get("poor_rollback_history"))),
                    bool_factor(bool(change.get("touches_schema"))),
                    bool_factor(bool(change.get("touches_infrastructure"))),
                ]
            ),
            3,
        ),
        "observability_coverage": round(observability_coverage, 3),
        "customer_exposure": round(customer_exposure, 3),
        "regulated_exposure": round(
            average(
                [
                    bool_factor(bool(service.get("regulated_zone"))),
                    1.0 if environment.get("compliance_zone") else 0.0,
                ]
            ),
            3,
        ),
    }
