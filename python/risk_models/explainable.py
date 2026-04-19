from __future__ import annotations

from analytics.history import analyze_change_pattern
from common.models import average, bool_factor, bounded_ratio, clamp, dedupe


def augment_risk(payload: dict) -> dict:
    change = payload["change"]
    service = payload["service"]
    environment = payload["environment"]
    pattern = analyze_change_pattern(payload)

    normalized_factors = {
        "file_surface": round(bounded_ratio(float(change.get("file_count", 0)), 50.0), 3),
        "resource_surface": round(
            bounded_ratio(float(change.get("resource_count", 0)), 10.0), 3
        ),
        "incident_history": round(
            bounded_ratio(float(change.get("historical_incident_count", 0)), 5.0), 3
        ),
        "dependency_surface": round(
            average(
                [
                    bool_factor(bool(change.get("dependency_changes"))),
                    bounded_ratio(float(service.get("dependent_services_count", 0)), 4.0),
                ]
            ),
            3,
        ),
        "identity_surface": round(
            average(
                [
                    bool_factor(bool(change.get("touches_iam"))),
                    bool_factor(bool(change.get("touches_secrets"))),
                ]
            ),
            3,
        ),
        "schema_surface": round(bool_factor(bool(change.get("touches_schema"))), 3),
        "customer_exposure": pattern["customer_exposure"],
        "regulated_exposure": pattern["regulated_exposure"],
        "observability_gap": round(1.0 - pattern["observability_coverage"], 3),
    }

    confidence_adjustment = 0.0
    confidence_adjustment -= normalized_factors["observability_gap"] * 0.18
    confidence_adjustment -= pattern["rollback_fragility"] * 0.1
    confidence_adjustment -= pattern["historical_incident_bias"] * 0.08
    confidence_adjustment += (1.0 - normalized_factors["file_surface"]) * 0.03
    confidence_adjustment = round(clamp(confidence_adjustment, -0.3, 0.12), 3)

    explanations = []
    guardrails = []

    if normalized_factors["identity_surface"] >= 0.5:
        explanations.append(
            "supplemental analysis classified the change as identity-sensitive and raised access-regression concern"
        )
        guardrails.extend(["privilege-regression-review", "token-and-secret-smoke-check"])
    if normalized_factors["schema_surface"] >= 1.0:
        explanations.append(
            "supplemental analysis detected schema sensitivity and increased rollback fragility weighting"
        )
        guardrails.extend(["migration-rehearsal", "backward-compatibility-check"])
    if normalized_factors["customer_exposure"] >= 0.66:
        explanations.append(
            "supplemental analysis detected meaningful customer-path exposure and recommends tighter verification"
        )
        guardrails.extend(["tenant-cohort-canary", "business-kpi-watch"])
    if normalized_factors["observability_gap"] >= 0.5:
        explanations.append(
            "supplemental analysis lowered confidence because observability or SLO coverage is incomplete"
        )
        guardrails.append("operator-observation-window")
    if pattern["volatility_index"] >= 0.6:
        explanations.append(
            "supplemental analysis classified the change as operationally volatile based on breadth and history"
        )
        guardrails.append("extended-canary-observation")

    guardrails.append(f"cluster:{pattern['cluster']}")

    return {
        "normalized_factors": normalized_factors,
        "confidence_adjustment": confidence_adjustment,
        "supplemental_explanations": dedupe(explanations),
        "recommended_guardrails": dedupe(guardrails),
        "change_cluster": pattern["cluster"],
        "historical_pattern": pattern,
    }
