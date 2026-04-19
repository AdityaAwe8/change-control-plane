from __future__ import annotations

from typing import Iterable, List


def clamp(value: float, lower: float, upper: float) -> float:
    return max(lower, min(upper, value))


def bounded_ratio(value: float, upper_bound: float) -> float:
    if upper_bound <= 0:
        return 0.0
    return clamp(value / upper_bound, 0.0, 1.0)


def bool_factor(value: bool) -> float:
    return 1.0 if value else 0.0


def average(values: Iterable[float]) -> float:
    items = list(values)
    if not items:
        return 0.0
    return sum(items) / len(items)


def dedupe(items: Iterable[str]) -> List[str]:
    result: List[str] = []
    seen = set()
    for item in items:
        normalized = item.strip()
        if not normalized or normalized in seen:
            continue
        seen.add(normalized)
        result.append(normalized)
    return result
