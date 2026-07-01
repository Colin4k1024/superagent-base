# tests/matrix/report.py
"""
Generate a Markdown matrix test report.

Run with: python tests/matrix/report.py
Runs pytest on all matrix tests and writes tests/matrix/REPORT.md.
"""
import subprocess
import sys
import json
from datetime import datetime
from pathlib import Path

REPORT_PATH = Path(__file__).parent / "REPORT.md"
BACKENDS = {"go": 8888, "python": 8889, "java": 8890}
REPO_ROOT = Path(__file__).parent.parent.parent


def run_pytest_json() -> dict:
    json_out = "/tmp/matrix-report.json"
    subprocess.run(
        [sys.executable, "-m", "pytest", "tests/matrix/",
         "--tb=short", "-q",
         "--json-report", f"--json-report-file={json_out}"],
        cwd=REPO_ROOT,
    )
    try:
        with open(json_out) as f:
            return json.load(f)
    except FileNotFoundError:
        return {"tests": [], "summary": {}}


def build_markdown(data: dict) -> str:
    now = datetime.now().strftime("%Y-%m-%d %H:%M")
    lines = [
        f"# Matrix Test Report — {now}\n",
        "| Test | Go :8888 | Python :8889 | Java :8890 |",
        "|---|:---:|:---:|:---:|",
    ]

    tests_by_name: dict[str, dict] = {}
    for t in data.get("tests", []):
        node = t["nodeid"]
        # Extract backend from parametrize id: test_foo[go] -> go
        backend = "unknown"
        for b in BACKENDS:
            if f"[{b}]" in node:
                backend = b
                break
        base = node.split("[")[0].split("::")[-1]
        tests_by_name.setdefault(base, {})[backend] = t["outcome"]

    for name, outcomes in sorted(tests_by_name.items()):
        go     = "✅" if outcomes.get("go")     == "passed" else ("❌" if "go"     in outcomes else "—")
        python = "✅" if outcomes.get("python")  == "passed" else ("❌" if "python" in outcomes else "—")
        java   = "✅" if outcomes.get("java")    == "passed" else ("❌" if "java"   in outcomes else "—")
        lines.append(f"| `{name}` | {go} | {python} | {java} |")

    summary = data.get("summary", {})
    passed  = summary.get("passed", 0)
    failed  = summary.get("failed", 0)
    total   = summary.get("total",  0)
    lines += ["", f"**Total: {passed}/{total} passed, {failed} failed**"]
    return "\n".join(lines) + "\n"


if __name__ == "__main__":
    print("Running matrix tests...")
    data = run_pytest_json()
    md = build_markdown(data)
    REPORT_PATH.write_text(md)
    print(f"Report written to {REPORT_PATH}")
    print(md)
