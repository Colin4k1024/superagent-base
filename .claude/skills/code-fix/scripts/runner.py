#!/usr/bin/env python3
"""
Code Fix Runner - Main orchestrator.

Coordinates the complete code-fix workflow.
No external dependencies - works completely offline.
"""

import argparse
import json
import os
import sys
import subprocess
from pathlib import Path
from typing import Any, Dict, Optional

# Add scripts directory to path
SCRIPT_DIR = Path(__file__).parent


def run_script(script_name: str, input_data: Dict) -> Dict:
    """Run a Python script and return its output."""
    script_path = SCRIPT_DIR / script_name

    try:
        result = subprocess.run(
            [sys.executable, str(script_path)],
            input=json.dumps(input_data),
            capture_output=True,
            text=True,
            timeout=30,
        )

        if result.returncode != 0:
            return {"error": result.stderr}

        return json.loads(result.stdout)
    except Exception as e:
        return {"error": str(e)}


def build_error_info(args) -> Dict[str, Any]:
    """Build error info from command line arguments."""
    error_info = {}

    if args.error_type:
        error_info["errorType"] = args.error_type

    if args.error_message:
        error_info["errorMessage"] = args.error_message

    if args.file:
        error_info["errorLocation"] = {"filePath": args.file}
        if args.line:
            error_info["errorLocation"]["lineNumber"] = args.line
        if args.column:
            error_info["errorLocation"]["columnNumber"] = args.column

    if args.stack_trace:
        error_info["stackTrace"] = args.stack_trace

    return error_info


def run_full_workflow(args) -> Dict[str, Any]:
    """Run the complete code-fix workflow."""
    # Get error info from args or stdin
    error_info = build_error_info(args)

    if not error_info.get("errorMessage"):
        print("Error: error-message is required", file=sys.stderr)
        return {"error": "errorMessage is required"}

    print(f"Step 1: Analyzing error...")
    analysis = run_script("error_analyzer.py", {"errorInfo": error_info})
    if analysis.get("error"):
        print(f"  Analysis failed: {analysis.get('error')}", file=sys.stderr)
        return {"error": f"Analysis failed: {analysis.get('error')}"}
    print(f"  Root cause: {analysis.get('rootCause', 'Unknown')[:80]}...")

    print(f"Step 2: Generating solutions...")
    solutions_result = run_script(
        "solution_generator.py", {"errorInfo": error_info, "analysis": analysis}
    )
    solutions = solutions_result.get("solutions", [])
    print(f"  Generated {len(solutions)} solutions")

    if not solutions:
        return {"error": "Failed to generate solutions"}

    print(f"Step 3: Evaluating solutions...")
    eval_result = run_script(
        "evaluator.py",
        {"solutions": solutions, "errorInfo": error_info, "analysis": analysis},
    )
    best_id = eval_result.get("bestSolutionId")
    print(f"  Best solution: {best_id}")

    # Find best solution
    best_solution = None
    for sol in solutions:
        if sol.get("solutionId") == best_id:
            best_solution = sol
            break

    if not best_solution:
        return {"error": "Could not find best solution"}

    print(f"Step 4: Executing fix...")
    workspace = args.workspace or os.getcwd()

    # Include errorInfo in solution so executor can access errorLocation for line numbers
    solution_with_context = {
        **best_solution,
        "errorLocation": error_info.get("errorLocation", {}),
    }
    exec_result = run_script(
        "executor.py",
        {
            "solution": solution_with_context,
            "workspaceRoot": workspace,
            "runTest": args.run_test,
        },
    )

    status = exec_result.get("executionStatus")
    files = exec_result.get("modifiedFiles", [])
    print(f"  Status: {status}")
    print(f"  Modified: {', '.join(files) if files else 'none'}")

    return exec_result


def run_direct_fix(args) -> Dict[str, Any]:
    """Run direct fix mode (skip analysis)."""
    error_info = build_error_info(args)

    if not error_info.get("errorMessage"):
        print("Error: error-message is required", file=sys.stderr)
        return {"error": "errorMessage is required"}

    # Create minimal solution directly
    location = error_info.get("errorLocation", {})
    file_path = location.get("filePath", "unknown")

    solution = {
        "solutionId": "direct-fix",
        "description": "Direct fix for: " + error_info.get("errorMessage", "")[:50],
        "filePath": file_path,
        "originalCode": "",
        "fixedCode": "# TODO: Fix based on error: "
        + error_info.get("errorMessage", ""),
    }

    workspace = args.workspace or os.getcwd()
    result = run_script(
        "executor.py",
        {"solution": solution, "workspaceRoot": workspace, "runTest": args.run_test},
    )

    return result


def main():
    parser = argparse.ArgumentParser(
        description="Code Fix - Fix code errors locally",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python runner.py --error-type TEST_FAILURE \\
    --error-message "Expected: 200, Actual: 404" \\
    --file src/app.py --line 42

  python runner.py --direct-fix --error-message "NullPointer" --file app.js

  python runner.py --step analyze --error-message "Error" --file app.py
        """,
    )

    # Error info arguments
    parser.add_argument(
        "--error-type",
        help="Error type: TEST_FAILURE, RUNTIME_ERROR, COMPILATION_ERROR",
    )
    parser.add_argument("--error-message", help="Error message")
    parser.add_argument("--file", help="File path where error occurred")
    parser.add_argument("--line", type=int, help="Line number")
    parser.add_argument("--column", type=int, help="Column number")
    parser.add_argument("--stack-trace", help="Stack trace")

    # Options
    parser.add_argument("--workspace", help="Workspace root directory")
    parser.add_argument("--run-test", action="store_true", help="Run tests after fix")
    parser.add_argument("--direct-fix", action="store_true", help="Direct fix mode")
    parser.add_argument(
        "--step",
        choices=["analyze", "generate", "evaluate", "execute"],
        help="Run specific step only",
    )
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")

    args = parser.parse_args()

    # Run workflow
    try:
        if args.step:
            error_info = build_error_info(args)
            if args.step == "analyze":
                result = run_script("error_analyzer.py", {"errorInfo": error_info})
            elif args.step == "generate":
                analysis = {"rootCause": "N/A"}
                result = run_script(
                    "solution_generator.py",
                    {"errorInfo": error_info, "analysis": analysis},
                )
            elif args.step == "evaluate":
                result = {"error": "Evaluate requires solutions from generate"}
            else:
                result = {"error": "Unknown step"}
        elif args.direct_fix:
            result = run_direct_fix(args)
        else:
            result = run_full_workflow(args)

        print(json.dumps(result, indent=2))

        if result.get("executionStatus") == "FAILED" or result.get("error"):
            sys.exit(1)

    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
