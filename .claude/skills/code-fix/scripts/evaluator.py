#!/usr/bin/env python3
"""
Solution Evaluator - Rule-based solution evaluation.

Evaluates and scores fix solutions using rule-based scoring.
No external dependencies - works completely offline.
"""

import json
import sys
from typing import Any, Dict, List


# Default scoring weights
DEFAULT_WEIGHTS = {
    "correctness": 0.4,
    "minimalChange": 0.3,
    "risk": 0.3,
}


def evaluate_solutions(
    solutions: List[Dict], error_info: Dict, analysis: Dict, weights: Dict = None
) -> Dict[str, Any]:
    """Evaluate solutions and select the best."""
    if weights is None:
        weights = DEFAULT_WEIGHTS

    if not solutions:
        return {
            "evaluations": [],
            "bestSolutionId": None,
            "error": "No solutions to evaluate",
        }

    evaluations = []
    for sol in solutions:
        eval_result = evaluate_single_solution(sol, error_info, analysis, weights)
        evaluations.append(eval_result)

    # Sort by total score (descending)
    evaluations.sort(key=lambda x: x["scores"]["total"], reverse=True)

    best = evaluations[0] if evaluations else None
    best_id = best["solutionId"] if best else None

    return {
        "evaluations": evaluations,
        "bestSolutionId": best_id,
        "bestScores": best["scores"] if best else None,
    }


def evaluate_single_solution(
    solution: Dict, error_info: Dict, analysis: Dict, weights: Dict
) -> Dict[str, Any]:
    """Evaluate a single solution."""
    scores = {}
    reasoning_parts = []

    # 1. Correctness score (0-1)
    correctness = evaluate_correctness(solution, error_info, analysis)
    scores["correctness"] = round(correctness, 2)
    reasoning_parts.append(f"Correctness: {correctness}")

    # 2. Minimal change score (0-1)
    minimal = evaluate_minimal_change(solution)
    scores["minimalChange"] = round(minimal, 2)
    reasoning_parts.append(f"Minimal change: {minimal}")

    # 3. Risk score (0-1, lower is better)
    risk = evaluate_risk(solution)
    scores["risk"] = round(risk, 2)
    reasoning_parts.append(f"Risk: {risk}")

    # Calculate weighted total
    total = (
        weights["correctness"] * correctness
        + weights["minimalChange"] * minimal
        + weights["risk"] * (1 - risk)  # invert risk
    )
    scores["total"] = round(total, 2)

    return {
        "solutionId": solution.get("solutionId", "unknown"),
        "scores": scores,
        "reasoning": "; ".join(reasoning_parts),
    }


def evaluate_correctness(solution: Dict, error_info: Dict, analysis: Dict) -> float:
    """Evaluate how likely the fix is correct."""
    description = solution.get("description", "").lower()
    fixed_code = solution.get("fixedCode", "").lower()
    root_cause = analysis.get("rootCause", "").lower()

    score = 0.5  # baseline

    # Check if description addresses root cause
    root_keywords = [w for w in root_cause.split() if len(w) > 4]
    matches = sum(1 for kw in root_keywords if kw in description or kw in fixed_code)
    if root_keywords:
        score = min(0.5 + (matches / len(root_keywords)) * 0.5, 1.0)

    # Boost for specific error type matches
    error_msg = error_info.get("errorMessage", "").lower()
    if "null" in error_msg and ("null" in description or "null" in fixed_code):
        score += 0.1
    if "undefined" in error_msg and ("undefined" in description or "?" in fixed_code):
        score += 0.1
    if "type" in error_msg and ("type" in description or "typeof" in fixed_code):
        score += 0.1

    return min(score, 1.0)


def evaluate_minimal_change(solution: Dict) -> float:
    """Evaluate if the fix is minimal."""
    fixed_code = solution.get("fixedCode", "")

    if not fixed_code:
        return 0.5

    # Count lines of code change
    lines = fixed_code.split("\n")
    line_count = len([l for l in lines if l.strip()])

    # Fewer lines = higher score
    if line_count <= 2:
        return 1.0
    elif line_count <= 5:
        return 0.8
    elif line_count <= 10:
        return 0.6
    elif line_count <= 20:
        return 0.4
    else:
        return 0.2


def evaluate_risk(solution: Dict) -> float:
    """Evaluate risk level of the fix."""
    affected = solution.get("affectedFiles", [])
    fixed_code = solution.get("fixedCode", "")

    risk = 0.3  # baseline

    # More affected files = higher risk
    risk += len(affected) * 0.1

    # Large changes = higher risk
    if fixed_code:
        length = len(fixed_code)
        if length > 500:
            risk += 0.3
        elif length > 200:
            risk += 0.2
        elif length > 100:
            risk += 0.1

    # High-risk patterns
    fixed_lower = fixed_code.lower()
    if "delete" in fixed_lower or "drop" in fixed_lower:
        risk += 0.3
    if "force" in fixed_lower or "!" in fixed_lower:
        risk += 0.1

    return min(risk, 1.0)


def main():
    if not sys.stdin.isatty():
        input_data = json.load(sys.stdin)
    else:
        input_data = {}

    solutions = input_data.get("solutions", [])
    error_info = input_data.get("errorInfo", {})
    analysis = input_data.get("analysis", {})
    weights = input_data.get("weights", DEFAULT_WEIGHTS)

    if not solutions:
        print(
            json.dumps(
                {
                    "evaluations": [],
                    "bestSolutionId": None,
                    "error": "No solutions provided",
                }
            )
        )
        sys.exit(1)

    result = evaluate_solutions(solutions, error_info, analysis, weights)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
