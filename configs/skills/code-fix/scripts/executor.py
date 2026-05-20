#!/usr/bin/env python3
"""
Fix Executor - Apply fixes to workspace files.

Applies the selected fix solution to workspace files.
No external dependencies - works completely offline.
"""

import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional


def find_file_in_workspace(workspace_root: str, file_path: str) -> Optional[Path]:
    """Find a file in the workspace."""
    workspace = Path(workspace_root)

    # Direct path
    direct = workspace / file_path
    if direct.exists():
        return direct

    # Try common source directories
    for base in ["src", "app", "lib", "source", "main", ""]:
        candidate = workspace / base / file_path if base else workspace / file_path
        if candidate.exists():
            return candidate

    return None


def read_file(path: Path) -> Optional[str]:
    """Read file content safely."""
    try:
        return path.read_text(encoding="utf-8")
    except Exception:
        return None


def write_file(path: Path, content: str) -> bool:
    """Write content to file safely."""
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        return True
    except Exception as e:
        print(f"Error writing file: {e}", file=sys.stderr)
        return False


def apply_fix(
    file_path: str,
    fixed_code: str,
    workspace_root: str,
    original_code: str = "",
    line_number: int = None,
) -> tuple[bool, str]:
    """Apply fix to a file.

    Strategy:
    1. If originalCode is provided and matches, replace it with fixedCode
    2. If line_number is provided, try to replace that line
    3. Otherwise, append suggestion as comment
    """
    target_file = find_file_in_workspace(workspace_root, file_path)

    if not target_file:
        return False, f"File not found in workspace: {file_path}"

    content = read_file(target_file)
    if content is None:
        return False, f"Cannot read file: {file_path}"

    new_content = None
    log = ""

    # Strategy 1: Replace by originalCode if it's a valid (not placeholder) match
    if original_code and original_code not in ["", "# code to replace", "// code"]:
        if original_code in content:
            new_content = content.replace(original_code, fixed_code, 1)
            log = f"Replaced original code in {file_path}"

    # Strategy 2: Replace by line number if available
    elif line_number is not None:
        lines = content.split("\n")
        if 0 < line_number <= len(lines):
            # Replace the specific line with fixed_code
            lines[line_number - 1] = fixed_code
            new_content = "\n".join(lines)
            log = f"Replaced line {line_number} in {file_path}"

    # Strategy 3: Fall back to appending suggestion
    if new_content is None:
        comment_prefix = (
            "// "
            if target_file.suffix in [".js", ".ts", ".java", ".c", ".cpp", ".go", ".rs"]
            else "# "
        )
        new_content = (
            content
            + f"\n\n{comment_prefix}Fix suggested:\n{comment_prefix}"
            + fixed_code.replace("\n", f"\n{comment_prefix}")
        )
        log = f"Appended fix suggestion to {file_path} (no exact match found)"

    if write_file(target_file, new_content):
        return True, log

    return False, f"Failed to write to {file_path}"


def run_tests(workspace_root: str) -> tuple[bool, str]:
    """Auto-detect and run tests."""
    workspace = Path(workspace_root)
    files = [f.name for f in workspace.iterdir()]

    test_commands = []

    if "package.json" in files:
        test_commands.append("npm test")
    if "pom.xml" in files:
        test_commands.append("mvn test")
    if "build.gradle" in files:
        test_commands.append("./gradlew test")
    if "requirements.txt" in files or "pyproject.toml" in files:
        test_commands.append("python -m pytest")
    if "go.mod" in files:
        test_commands.append("go test ./...")
    if "Cargo.toml" in files:
        test_commands.append("cargo test")

    for cmd in test_commands:
        try:
            result = subprocess.run(
                cmd,
                shell=True,
                cwd=workspace_root,
                capture_output=True,
                text=True,
                timeout=120,
            )
            return result.returncode == 0, result.stdout + result.stderr
        except Exception:
            continue

    return False, "No test command found"


def execute_fix(
    solution: Dict[str, Any], workspace_root: str, run_test: bool = False
) -> Dict[str, Any]:
    """Execute the fix solution."""
    file_path = solution.get("filePath", "")
    fixed_code = solution.get("fixedCode", "")
    original_code = solution.get("originalCode", "")

    # Extract line number from error location if available
    error_location = solution.get("errorLocation", {})
    line_number = (
        error_location.get("lineNumber") if isinstance(error_location, dict) else None
    )

    if not file_path or not fixed_code:
        return {
            "executionStatus": "FAILED",
            "modifiedFiles": [],
            "executionLog": "Missing filePath or fixedCode",
            "newErrors": [],
        }

    success, log = apply_fix(
        file_path, fixed_code, workspace_root, original_code, line_number
    )

    if not success:
        return {
            "executionStatus": "FAILED",
            "modifiedFiles": [],
            "executionLog": log,
            "newErrors": [],
        }

    modified_files = [file_path]
    execution_log = log

    # Run tests if requested
    if run_test:
        test_success, test_output = run_tests(workspace_root)
        execution_log += f"\n\n--- Test Results ---\n{test_output}"

        if not test_success:
            return {
                "executionStatus": "FAILED",
                "modifiedFiles": modified_files,
                "executionLog": execution_log,
                "newErrors": [{"errorMessage": "Tests failed after fix"}],
            }

    return {
        "executionStatus": "SUCCESS",
        "modifiedFiles": modified_files,
        "executionLog": execution_log,
        "newErrors": [],
    }


def main():
    if not sys.stdin.isatty():
        input_data = json.load(sys.stdin)
    else:
        input_data = {}

    solution = input_data.get("solution", {})
    workspace_root = input_data.get("workspaceRoot", os.getcwd())
    run_test = input_data.get("runTest", False)

    if not solution:
        print(
            json.dumps(
                {
                    "executionStatus": "FAILED",
                    "modifiedFiles": [],
                    "executionLog": "No solution provided",
                    "newErrors": [],
                }
            )
        )
        sys.exit(1)

    result = execute_fix(solution, workspace_root, run_test)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
