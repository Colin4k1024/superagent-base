#!/usr/bin/env python3
"""
Error Analyzer - Rule-based error analysis.

Parses error information and extracts root cause using pattern matching.
No external dependencies - works completely offline.
"""

import json
import re
import sys
from pathlib import Path
from typing import Any, Dict, Optional


# Error pattern database
ERROR_PATTERNS = {
    "NullPointerException": {
        "root_cause": "Object is null when accessed",
        "fix_direction": "Add null check before access",
    },
    "NullPointerError": {
        "root_cause": "Variable is null or undefined",
        "fix_direction": "Add null/undefined check",
    },
    "Cannot read property": {
        "root_cause": "Property access on undefined/null",
        "fix_direction": "Add optional chaining or null check",
    },
    "undefined is not": {
        "root_cause": "Using undefined value",
        "fix_direction": "Add initialization or null check",
    },
    "is not a function": {
        "root_cause": "Calling non-function or undefined",
        "fix_direction": "Check function exists before calling",
    },
    "TypeError": {
        "root_cause": "Type mismatch",
        "fix_direction": "Check variable type or add conversion",
    },
    "SyntaxError": {
        "root_cause": "Syntax error in code",
        "fix_direction": "Fix syntax per error message",
    },
    "ReferenceError": {
        "root_cause": "Using undefined variable",
        "fix_direction": "Declare variable or import",
    },
    "AssertionError": {
        "root_cause": "Assertion failed - expected != actual",
        "fix_direction": "Fix expected value or actual logic",
    },
    "Expected": {
        "root_cause": "Test assertion failed",
        "fix_direction": "Fix expected value or implementation",
    },
    "404": {
        "root_cause": "Resource not found",
        "fix_direction": "Check URL or create resource",
    },
    "500": {
        "root_cause": "Server error",
        "fix_direction": "Check server logs and fix backend",
    },
    "connection refused": {
        "root_cause": "Service not running or unreachable",
        "fix_direction": "Start service or check URL",
    },
    "ENOENT": {
        "root_cause": "File or directory not found",
        "fix_direction": "Check file path or create file",
    },
    "Module not found": {
        "root_cause": "Import or require failed",
        "fix_direction": "Install module or fix import path",
    },
    "ImportError": {
        "root_cause": "Import failed",
        "fix_direction": "Fix import statement or install package",
    },
    "Duplicate local variable": {
        "root_cause": "Variable is declared multiple times in the same scope",
        "fix_direction": "Remove duplicate declaration or rename variable",
    },
    "already defined": {
        "root_cause": "Variable or function already defined",
        "fix_direction": "Remove duplicate declaration or rename",
    },
    "is already defined": {
        "root_cause": "Symbol already defined in current scope",
        "fix_direction": "Rename or remove duplicate definition",
    },
}


def parse_error_location(error_text: str) -> Dict[str, Any]:
    """Extract file path and line number from error message."""
    result = {"filePath": None, "lineNumber": None, "columnNumber": None}

    # Pattern: at file.js:line:col
    match = re.search(r"(?:at\s+)?([^\s]+):(\d+):(\d+)", error_text)
    if match:
        result["filePath"] = match.group(1)
        result["lineNumber"] = int(match.group(2))
        result["columnNumber"] = int(match.group(3))
        return result

    # Pattern: file.js:line
    match = re.search(r"(?:at\s+)?([^\s]+):(\d+)", error_text)
    if match:
        result["filePath"] = match.group(1)
        result["lineNumber"] = int(match.group(2))
        return result

    # Pattern: "file", line N
    match = re.search(r'["\']([^"\']+)["\'],\s*line\s+(\d+)', error_text, re.IGNORECASE)
    if match:
        result["filePath"] = match.group(1)
        result["lineNumber"] = int(match.group(2))
        return result

    return result


def detect_error_type(error_message: str) -> str:
    """Detect error type from error message."""
    msg_lower = error_message.lower()

    if any(p in msg_lower for p in ["null", "undefined", "cannot read", "is not"]):
        return "RUNTIME_ERROR"
    if any(p in msg_lower for p in ["expected", "assertion", "test"]):
        return "TEST_FAILURE"
    if any(p in msg_lower for p in ["syntax", "parse"]):
        return "COMPILATION_ERROR"
    if any(
        p in msg_lower for p in ["404", "500", "connection", "not found", "refused"]
    ):
        return "RUNTIME_ERROR"
    if any(p in msg_lower for p in ["duplicate", "already defined", "redefined"]):
        return "COMPILATION_ERROR"

    return "UNKNOWN"


def find_root_cause(error_message: str, stack_trace: str = "") -> str:
    """Find root cause from error message."""
    combined = error_message + " " + (stack_trace or "")

    for pattern, info in ERROR_PATTERNS.items():
        if pattern.lower() in combined.lower():
            return info["root_cause"]

    # Default analysis
    if "at line" in combined.lower():
        return "Error at specific line - check code at that location"
    if "in" in combined:
        return "Check error location for issue"
    return "Analyze error message for root cause"


def suggest_fix_directions(error_message: str, stack_trace: str = "") -> list:
    """Suggest fix directions based on error patterns."""
    combined = error_message + " " + (stack_trace or "")
    suggestions = []

    for pattern, info in ERROR_PATTERNS.items():
        if pattern.lower() in combined.lower():
            suggestions.append(info["fix_direction"])

    if not suggestions:
        suggestions.append("Analyze error message and fix accordingly")

    return suggestions


def analyze_error(error_info: Dict[str, Any]) -> Dict[str, Any]:
    """Main analysis function."""
    error_message = error_info.get("errorMessage", "")
    stack_trace = error_info.get("stackTrace", "")
    provided_location = error_info.get("errorLocation", {})

    # Detect error type
    error_type = error_info.get("errorType")
    if not error_type or error_type == "UNKNOWN":
        error_type = detect_error_type(error_message)

    # Parse location if not provided
    location = provided_location
    if not location or not location.get("filePath"):
        location = parse_error_location(error_message + " " + stack_trace)

    # Find root cause
    root_cause = find_root_cause(error_message, stack_trace)

    # Suggest fix directions
    fix_directions = suggest_fix_directions(error_message, stack_trace)

    return {
        "errorType": error_type,
        "rootCause": root_cause,
        "suggestedDirections": fix_directions,
        "errorLocation": location,
        "confidence": 0.85,
    }


def main():
    # Read input
    if not sys.stdin.isatty():
        input_data = json.load(sys.stdin)
    else:
        # Try to parse command line args as JSON
        input_data = {}

    error_info = input_data.get("errorInfo", input_data)

    if not error_info.get("errorMessage"):
        print(
            json.dumps(
                {
                    "error": "errorMessage is required",
                    "rootCause": "No error message provided",
                }
            )
        )
        sys.exit(1)

    result = analyze_error(error_info)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
