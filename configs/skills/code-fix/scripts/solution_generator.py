#!/usr/bin/env python3
"""
Solution Generator - Template-based fix solution generation.

Generates fix solutions based on error patterns and templates.
No external dependencies - works completely offline.
"""

import json
import re
import sys
from pathlib import Path
from typing import Any, Dict, List


# Fix solution templates
FIX_TEMPLATES = {
    "null_check": {
        "description": "Add null/undefined check before accessing property",
        "originalCode": "# existing code",
        "fixedCode": "# Check if value exists\nif (value != null && value != undefined) {\n    # existing code\n}",
        "risk": "low",
    },
    "optional_chaining": {
        "description": "Use optional chaining for safe property access",
        "originalCode": "object.property",
        "fixedCode": "object?.property",
        "risk": "low",
    },
    "null_coalescing": {
        "description": "Add null coalescing for default value",
        "originalCode": "value",
        "fixedCode": "value ?? defaultValue",
        "risk": "low",
    },
    "guard_clause": {
        "description": "Add early return guard clause",
        "originalCode": "function() {\n    # code\n}",
        "fixedCode": "function() {\n    if (!condition) return;\n    # code\n}",
        "risk": "low",
    },
    "try_catch": {
        "description": "Wrap code in try-catch for error handling",
        "originalCode": "# code that might throw",
        "fixedCode": "try {\n    # code that might throw\n} catch (error) {\n    console.error(error);\n}",
        "risk": "medium",
    },
    "type_check": {
        "description": "Add type check before operation",
        "originalCode": "# code",
        "fixedCode": "if (typeof value === 'expectedType') {\n    # code\n}",
        "risk": "low",
    },
    "import_add": {
        "description": "Add missing import statement",
        "originalCode": "",
        "fixedCode": "import { MissingModule } from 'module-name';",
        "risk": "low",
    },
    "return_check": {
        "description": "Add return value check",
        "originalCode": "function() {\n    result = compute()\n}",
        "fixedCode": "function() {\n    result = compute()\n    if (!result) return;\n}",
        "risk": "low",
    },
    "await_add": {
        "description": "Add await for async operation",
        "originalCode": "promise.then()",
        "fixedCode": "await promise",
        "risk": "medium",
    },
    "url_check": {
        "description": "Check URL validity before request",
        "originalCode": "fetch(url)",
        "fixedCode": "if (!url) throw new Error('URL required');\nfetch(url)",
        "risk": "low",
    },
}


# Language-specific patterns
LANGUAGE_PATTERNS = {
    "python": {
        "null_check": {
            "originalCode": "# code",
            "fixedCode": "if value is not None:\n    # code",
        },
        "guard_clause": {
            "originalCode": "def function():\n    code",
            "fixedCode": "def function():\n    if not condition:\n        return\n    code",
        },
    },
    "java": {
        "null_check": {
            "originalCode": "// code",
            "fixedCode": "if (value != null) {\n    // code\n}",
        },
    },
    "javascript": {
        "null_check": {
            "originalCode": "// code",
            "fixedCode": "if (value != null && value != undefined) {\n    // code\n}",
        },
    },
    "typescript": {
        "null_check": {
            "originalCode": "// code",
            "fixedCode": "if (value != null) {\n    // code\n}",
        },
    },
}


def detect_language(file_path: str) -> str:
    """Detect programming language from file extension."""
    ext = Path(file_path).suffix.lower() if hasattr(Path(file_path), "suffix") else ""

    mapping = {
        ".py": "python",
        ".js": "javascript",
        ".ts": "typescript",
        ".java": "java",
        ".go": "go",
        ".rs": "rust",
        ".rb": "ruby",
        ".php": "php",
        ".cs": "csharp",
    }
    return mapping.get(ext, "javascript")


def generate_solutions(error_info: Dict, analysis: Dict) -> Dict[str, Any]:
    """Generate fix solutions based on error and analysis."""
    error_type = error_info.get("errorType", "UNKNOWN")
    error_message = error_info.get("errorMessage", "")
    root_cause = analysis.get("rootCause", "")
    location = error_info.get("errorLocation", {})
    file_path = location.get("filePath", "unknown")

    language = detect_language(file_path)

    solutions = []

    # Generate solutions based on error patterns
    msg_lower = error_message.lower()

    # Null/undefined errors
    if any(
        p in msg_lower
        for p in ["null", "undefined", "cannot read", "cannot read property"]
    ):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Add null check",
                    file_path,
                    "if (value != null) { /* code */ }",
                    "value != null && value != undefined",
                ),
                create_solution(
                    "solution-2",
                    "Use optional chaining",
                    file_path,
                    "object?.property",
                    "Optional chaining",
                ),
                create_solution(
                    "solution-3",
                    "Add default value",
                    file_path,
                    "value ?? defaultValue",
                    "Null coalescing",
                ),
            ]
        )

    # Test failure / assertion errors
    elif any(p in msg_lower for p in ["expected", "assertion", "test failed"]):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Fix expected value in test",
                    file_path,
                    "expect(actual).toBe(expected)",
                    "Check test expectations match implementation",
                ),
                create_solution(
                    "solution-2",
                    "Fix implementation to match expectation",
                    file_path,
                    "return correctValue",
                    "Fix implementation logic",
                ),
                create_solution(
                    "solution-3",
                    "Add logging to debug",
                    file_path,
                    "console.log(debugInfo)",
                    "Add debug logging",
                ),
            ]
        )

    # Not found errors (404, file not found)
    elif any(p in msg_lower for p in ["404", "not found", "enoent", "not found"]):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Check resource URL/path",
                    file_path,
                    "Verify resource exists",
                    "Check URL or file path is correct",
                ),
                create_solution(
                    "solution-2",
                    "Create missing resource",
                    file_path,
                    "Create the missing file/endpoint",
                    "Create missing resource",
                ),
                create_solution(
                    "solution-3",
                    "Add error handling",
                    file_path,
                    "try { /* code */ } catch { /* handle */ }",
                    "Handle missing resource gracefully",
                ),
            ]
        )

    # Type errors
    elif any(p in msg_lower for p in ["typeerror", "type error", "is not a"]):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Add type conversion",
                    file_path,
                    "String(value)",
                    "Convert to expected type",
                ),
                create_solution(
                    "solution-2",
                    "Add type check",
                    file_path,
                    "if (typeof value === 'string')",
                    "Check type before operation",
                ),
                create_solution(
                    "solution-3",
                    "Use type assertion",
                    file_path,
                    "value as ExpectedType",
                    "Assert correct type",
                ),
            ]
        )

    # Reference errors
    elif any(
        p in msg_lower for p in ["referenceerror", "is not defined", "cannot find"]
    ):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Check variable declaration",
                    file_path,
                    "const/let/var variableName",
                    "Declare missing variable",
                ),
                create_solution(
                    "solution-2",
                    "Fix import statement",
                    file_path,
                    "import { name } from 'module'",
                    "Add or fix import",
                ),
                create_solution(
                    "solution-3",
                    "Check module installation",
                    file_path,
                    "npm install module-name",
                    "Install missing dependency",
                ),
            ]
        )

    # Syntax errors
    elif any(p in msg_lower for p in ["syntaxerror", "syntax error"]):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Fix syntax per error message",
                    file_path,
                    "# fix syntax",
                    "Fix syntax as indicated in error",
                ),
                create_solution(
                    "solution-2",
                    "Check for missing characters",
                    file_path,
                    "Check brackets, parentheses, quotes",
                    "Add missing characters",
                ),
                create_solution(
                    "solution-3",
                    "Remove invalid characters",
                    file_path,
                    "Remove invalid syntax",
                    "Clean up code",
                ),
            ]
        )

    # Connection errors
    elif any(p in msg_lower for p in ["connection", "refused", "timeout"]):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Check service availability",
                    file_path,
                    "Verify service is running",
                    "Start or check service",
                ),
                create_solution(
                    "solution-2",
                    "Add retry logic",
                    file_path,
                    "retry(3)",
                    "Add retry with backoff",
                ),
                create_solution(
                    "solution-3",
                    "Add error handling",
                    file_path,
                    "try/catch with fallback",
                    "Handle connection failure gracefully",
                ),
            ]
        )

    # Duplicate variable / already defined errors
    elif any(p in msg_lower for p in ["duplicate", "already defined", "redefined"]):
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Remove duplicate declaration",
                    file_path,
                    "Remove the duplicate variable declaration",
                    "Delete the second declaration of the same variable",
                ),
                create_solution(
                    "solution-2",
                    "Rename variable",
                    file_path,
                    "Rename one of the variables to a unique name",
                    "Use a different name for one of the variables",
                ),
                create_solution(
                    "solution-3",
                    "Check scope/merge declarations",
                    file_path,
                    "Merge declarations if in same scope, or separate scopes",
                    "Consolidate or separate variable declarations",
                ),
            ]
        )

    # Default fallback solutions
    else:
        solutions.extend(
            [
                create_solution(
                    "solution-1",
                    "Analyze and fix based on error",
                    file_path,
                    "Review error location",
                    "Analyze error message",
                ),
                create_solution(
                    "solution-2",
                    "Check recent changes",
                    file_path,
                    "git diff",
                    "Check what changed",
                ),
                create_solution(
                    "solution-3",
                    "Add error handling",
                    file_path,
                    "try/catch/finally",
                    "Handle error gracefully",
                ),
            ]
        )

    return {"solutions": solutions}


def create_solution(
    solution_id: str,
    description: str,
    file_path: str,
    fixed_code: str,
    pre_simulation: str,
) -> Dict[str, Any]:
    """Create a solution object."""
    return {
        "solutionId": solution_id,
        "description": description,
        "filePath": file_path,
        "originalCode": "# code to replace",
        "fixedCode": fixed_code,
        "affectedFiles": [],
        "preSimulation": pre_simulation,
    }


def main():
    if not sys.stdin.isatty():
        input_data = json.load(sys.stdin)
    else:
        input_data = {}

    error_info = input_data.get("errorInfo", {})
    analysis = input_data.get("analysis", {})

    if not error_info.get("errorMessage"):
        print(json.dumps({"solutions": [], "error": "errorMessage is required"}))
        sys.exit(1)

    result = generate_solutions(error_info, analysis)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
