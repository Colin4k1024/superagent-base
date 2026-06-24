#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Superagent Three-Base E2E Parity Verification            ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check Python
if ! command -v python3 &>/dev/null; then
    echo -e "${RED}Error: python3 not found${NC}"
    exit 1
fi

# Install httpx if needed
pip3 install httpx -q 2>/dev/null || pip install httpx -q

# Check all three bases
echo -e "${YELLOW}Checking bases...${NC}"

GO_OK=false
PY_OK=false
JAVA_OK=false

if curl -s -o /dev/null -w "%{http_code}" "http://localhost:8888/health" 2>/dev/null | grep -q "200"; then
    echo -e "  Go (8888): ${GREEN}OK${NC}"
    GO_OK=true
else
    echo -e "  Go (8888): ${RED}NOT RUNNING${NC}"
fi

if curl -s -o /dev/null -w "%{http_code}" "http://localhost:8889/health" 2>/dev/null | grep -q "200"; then
    echo -e "  Python (8889): ${GREEN}OK${NC}"
    PY_OK=true
else
    echo -e "  Python (8889): ${RED}NOT RUNNING${NC}"
fi

if curl -s -o /dev/null -w "%{http_code}" "http://localhost:8890/health" 2>/dev/null | grep -q "200"; then
    echo -e "  Java (8890): ${GREEN}OK${NC}"
    JAVA_OK=true
else
    echo -e "  Java (8890): ${RED}NOT RUNNING${NC}"
fi

if [ "$GO_OK" != "true" ] || [ "$PY_OK" != "true" ] || [ "$JAVA_OK" != "true" ]; then
    echo ""
    echo -e "${RED}Please start all three bases before running E2E tests.${NC}"
    echo ""
    echo "  Go:     cd backend && make dev-server"
    echo "  Python: cd python && uvicorn superagent.server:app --port 8889"
    echo "  Java:   cd java && mvn spring-boot:run"
    exit 1
fi

echo ""
echo -e "${GREEN}Running E2E parity tests...${NC}"
echo ""

# Run the Python test suite
python3 tests/e2e/test_parity.py

EXIT_CODE=$?

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   ALL TESTS PASSED — Three bases have full parity         ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
else
    echo -e "${RED}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║   SOME TESTS FAILED — Parity issues detected              ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════╝${NC}"
fi

echo ""
echo -e "Report: ${BLUE}tests/e2e/E2E_PARITY_REPORT.md${NC}"
echo ""

exit $EXIT_CODE
