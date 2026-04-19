from __future__ import annotations

from analytics.history import analyze_change_pattern
from common.models import dedupe


def simulate_rollout(payload: dict) -> dict:
    change = payload["change"]
    service = payload["service"]
    environment = payload["environment"]
    assessment = payload["assessment"]
    plan = payload["plan"]
    pattern = analyze_change_pattern(payload)

    hotspots = []
    verification_focus = list(plan.get("verification_signals", []))
    timeline_notes = []
    failure_modes = []

    if change.get("touches_infrastructure"):
        hotspots.append("infrastructure drift or dependency readiness can block promotion")
        failure_modes.append("environment parity regression")
    if change.get("touches_schema"):
        hotspots.append("schema compatibility must remain safe across old and new versions")
        verification_focus.append("migration-compatibility")
        failure_modes.append("partial rollback incompatibility")
    if change.get("touches_iam") or change.get("touches_secrets"):
        hotspots.append("identity and secret changes can fail after deploy despite green unit tests")
        verification_focus.append("access-regression")
        failure_modes.append("runtime authorization failure")
    if service.get("customer_facing"):
        verification_focus.extend(["customer-journey-health", "premium-tenant-health"])
        failure_modes.append("customer-path degradation")
    if environment.get("production"):
        timeline_notes.append("maintain an observation window before promotion because production exposure is active")
    if plan.get("strategy") == "canary":
        timeline_notes.append("hold the canary long enough to compare latency, error rate, and business signals")
    if plan.get("strategy") == "phased-rollout":
        timeline_notes.append("promote only after each cohort remains stable through a full verification interval")

    recommended_next_action = "proceed_with_guardrails"
    if assessment.get("level") == "critical":
        recommended_next_action = "manual_review_required"
    elif plan.get("strategy") == "canary":
        recommended_next_action = "canary_observe_longer"
    elif plan.get("strategy") == "phased-rollout":
        recommended_next_action = "advance_by_cohort"

    return {
        "recommended_next_action": recommended_next_action,
        "risk_hotspots": dedupe(hotspots),
        "timeline_notes": dedupe(timeline_notes),
        "verification_focus": dedupe(verification_focus),
        "metadata": {
            "cluster": pattern["cluster"],
            "failure_modes": dedupe(failure_modes),
            "observation_window_minutes": 30 if environment.get("production") else 10,
            "promotion_bias": "safety-first",
            "volatility_index": pattern["volatility_index"],
        },
    }
