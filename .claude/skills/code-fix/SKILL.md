---
name: code-fix
description: Analyzes code errors, generates fix solutions, evaluates and applies fixes. Use when fixing test failures, compilation errors, runtime exceptions, "编码修复", "code fix", "fix this error", or when the user asks to debug and fix code issues.
---

# Code Fix

## Overview

This skill provides a complete code-fix workflow: validate error, analyze root cause, generate multiple fix solutions, evaluate and select the best, then apply the fix. Works locally without any external dependencies.

## When to Use

Use this skill when:

- **Test failure**: Fix code after unit test, integration test, or E2E test failure
- **Compilation error**: Fix compile errors, syntax errors, type errors
- **Runtime error**: Fix exceptions, NullPointerError, runtime crashes
- **Explicit request**: User asks for "编码修复", "code fix", "fix this bug", "fix the error"
- **Debug sessions**: When debugging and root cause is found, proceed to fix

## Workflow

```mermaid
flowchart LR
  START --> validate
  validate --> analyze
  analyze --> generate
  generate --> evaluate
  evaluate --> execute
  execute --> END
```

### Steps

1. **Validate**: Ensure error info has required fields (message, location) - done by runner before calling analyze
2. **Analyze**: Parse error, extract root cause from stack trace
3. **Generate**: Create 3 fix solutions based on error patterns
4. **Evaluate**: Score solutions by correctness, risk, scope
5. **Execute**: Apply best fix to workspace files

## Quick Start

**Important**: Run scripts from the skill directory (where runner.py is located):

```bash
# From skill directory: .cursor/skills/code-fix/
cd .cursor/skills/code-fix

# Run full fix workflow
python scripts/runner.py --error-type TEST_FAILURE \
  --error-message "Expected: 200, Actual: 404" \
  --file src/app.py --line 42

# Or from project root with explicit workspace
python .cursor/skills/code-fix/scripts/runner.py \
  --error-type COMPILATION_ERROR \
  --error-message "Duplicate local variable state" \
  --file src/main/java/Example.java --line 54 \
  --workspace /path/to/project
```

**Arguments:**

- `--workspace`: Set workspace root (default: current directory)
- `--run-test`: Run tests after fix

## Input Format

```json
{
  "errorType": "TEST_FAILURE|RUNTIME_ERROR|COMPILATION_ERROR",
  "errorMessage": "The error message",
  "errorLocation": {
    "filePath": "src/app.py",
    "lineNumber": 42,
    "columnNumber": 10
  },
  "stackTrace": "Full stack trace (optional)"
}
```

## Output Format

```json
{
  "executionStatus": "SUCCESS|FAILED",
  "modifiedFiles": ["src/app.py"],
  "executionLog": "Applied fix to..."
}
```

## Scripts

| Script                                                 | Purpose                         |
| ------------------------------------------------------ | ------------------------------- |
| [runner.py](scripts/runner.py)                         | Main orchestrator               |
| [error_analyzer.py](scripts/error_analyzer.py)         | Parse error, extract root cause |
| [solution_generator.py](scripts/solution_generator.py) | Generate fix solutions          |
| [evaluator.py](scripts/evaluator.py)                   | Score and select best solution  |
| [executor.py](scripts/executor.py)                     | Apply fix to files              |

## Examples

### Example 1: Fix Test Failure

**Input:**

```
errorType: TEST_FAILURE
errorMessage: Expected: 200, Actual: 404
filePath: src/api/user.py
lineNumber: 42
```

**Process:**

1. Analyze: Root cause - API returns 404, missing user
2. Generate 3 solutions:
   - Add null check before API call
   - Add default user handling
   - Fix API endpoint URL
3. Evaluate: Solution 1 has best correctness/risk ratio
4. Execute: Apply null check

### Example 2: Fix Runtime Error

**Input:**

```
errorType: RUNTIME_ERROR
errorMessage: Cannot read property 'map' of undefined
stackTrace: at UserList.js:42:15
```

**Process:**

1. Analyze: Root cause - data not initialized
2. Generate solutions with guard clauses
3. Execute fix with null guard

## Error Patterns

The skill recognizes common error patterns:

| Pattern            | Error Types       | Typical Fix                |
| ------------------ | ----------------- | -------------------------- |
| Null/undefined     | RUNTIME_ERROR     | Add null check             |
| Type mismatch      | COMPILATION_ERROR | Cast or convert type       |
| Missing import     | COMPILATION_ERROR | Add import statement       |
| Async/await        | RUNTIME_ERROR     | Add await or .then()       |
| Route not found    | TEST_FAILURE      | Check URL or add route     |
| Assertion failed   | TEST_FAILURE      | Fix expected value         |
| Duplicate variable | COMPILATION_ERROR | Remove duplicate or rename |

## Running in Cursor

When triggered in Cursor, the skill:

1. Extracts error info from user message or test output
2. Runs the fix workflow using local scripts
3. **Applies changes to workspace files** (Agent edits directly based on workflow output)
4. Reports results

**Execution responsibility:**

- **In Cursor**: The Agent directly edits workspace files based on the workflow output (best solution, fixedCode, location). The executor script provides suggestions.
- **Via CLI**: Run `python scripts/runner.py` from the skill directory. The executor script attempts to replace code or append suggestions.

No external API or service required.

## Configuration

Works with zero config. Optional:

- `--workspace`: Set workspace root (default: current directory)
- `--run-test`: Run tests after fix
- `--verbose`: Show detailed progress

For detailed workflow diagram, see [references/workflow.md](references/workflow.md).
