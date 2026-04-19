#!/usr/bin/env python3
from __future__ import annotations

import json
import sys

from risk_models.explainable import augment_risk
from simulation.rollout import simulate_rollout


def main() -> int:
    if len(sys.argv) != 2:
        sys.stderr.write("usage: intelligence_cli.py <risk-augment|rollout-simulate>\n")
        return 2

    command = sys.argv[1].strip()
    try:
        payload = json.load(sys.stdin)
        if command == "risk-augment":
            result = augment_risk(payload)
        elif command == "rollout-simulate":
            result = simulate_rollout(payload)
        else:
            sys.stderr.write(f"unsupported command: {command}\n")
            return 2
    except Exception as exc:  # pragma: no cover - exercised by subprocess contract
        sys.stderr.write(f"{type(exc).__name__}: {exc}\n")
        return 1

    json.dump(result, sys.stdout, separators=(",", ":"), sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
