from __future__ import annotations

import json
import subprocess
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from risk_models.explainable import augment_risk
from simulation.rollout import simulate_rollout


def sample_payload() -> dict:
    return {
        "change": {
            "id": "chg_123",
            "organization_id": "org_123",
            "project_id": "proj_123",
            "service_id": "svc_123",
            "environment_id": "env_123",
            "summary": "update payment routing",
            "change_types": ["code", "iam"],
            "file_count": 12,
            "resource_count": 2,
            "touches_infrastructure": True,
            "touches_iam": True,
            "touches_secrets": False,
            "touches_schema": False,
            "dependency_changes": True,
            "historical_incident_count": 2,
            "poor_rollback_history": False,
        },
        "service": {
            "id": "svc_123",
            "organization_id": "org_123",
            "project_id": "proj_123",
            "team_id": "team_123",
            "name": "Checkout",
            "criticality": "mission_critical",
            "customer_facing": True,
            "has_slo": True,
            "has_observability": True,
            "regulated_zone": False,
            "dependent_services_count": 2,
        },
        "environment": {
            "id": "env_123",
            "organization_id": "org_123",
            "project_id": "proj_123",
            "name": "Production",
            "type": "production",
            "production": True,
            "compliance_zone": "",
        },
        "assessment": {
            "id": "risk_123",
            "score": 72,
            "level": "high",
            "confidence_score": 0.82,
            "recommended_rollout_strategy": "canary",
            "recommended_guardrails": ["health-check-gates"],
        },
        "plan": {
            "id": "roll_123",
            "strategy": "canary",
            "approval_required": True,
            "verification_signals": ["latency", "error-rate"],
        },
    }


class IntelligenceTests(unittest.TestCase):
    def test_augment_risk_returns_explainable_fields(self) -> None:
        result = augment_risk(sample_payload())
        self.assertIn("normalized_factors", result)
        self.assertIn("confidence_adjustment", result)
        self.assertIn("change_cluster", result)
        self.assertTrue(result["supplemental_explanations"])
        self.assertIn("cluster:", " ".join(result["recommended_guardrails"]))

    def test_simulate_rollout_returns_structured_guidance(self) -> None:
        result = simulate_rollout(sample_payload())
        self.assertIn(result["recommended_next_action"], {"canary_observe_longer", "advance_by_cohort", "manual_review_required", "proceed_with_guardrails"})
        self.assertTrue(result["verification_focus"])
        self.assertIn("metadata", result)
        self.assertIn("failure_modes", result["metadata"])

    def test_cli_contract_outputs_json(self) -> None:
        payload = json.dumps(sample_payload()).encode("utf-8")
        script = ROOT / "intelligence_cli.py"
        proc = subprocess.run(
            [sys.executable, str(script), "risk-augment"],
            input=payload,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=True,
        )
        parsed = json.loads(proc.stdout.decode("utf-8"))
        self.assertIn("normalized_factors", parsed)


if __name__ == "__main__":
    unittest.main()
