#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Superagent-Base E2E Test Runner${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Check Python
if ! command -v python3 &>/dev/null; then
    echo -e "${RED}Error: python3 not found${NC}"
    exit 1
fi

# Install dependencies
echo -e "${YELLOW}Installing dependencies...${NC}"
pip3 install -r requirements.txt -q 2>/dev/null || pip install -r requirements.txt -q

# Check services
echo -e "${YELLOW}Checking services...${NC}"

LLM_URL="${E2E_LLM_URL:-http://localhost:8000}"
BACKEND_URL="${E2E_BASE_URL:-http://localhost:8888}"

if curl -s "$LLM_URL/v1/models" -H "Authorization: Bearer ${E2E_LLM_API_KEY:-123456}" >/dev/null 2>&1; then
    echo -e "  LLM ($LLM_URL): ${GREEN}OK${NC}"
else
    echo -e "  LLM ($LLM_URL): ${RED}NOT REACHABLE${NC}"
    echo -e "${RED}Please start the LLM service first.${NC}"
    exit 1
fi

if curl -s "$BACKEND_URL/api/v1/agents" >/dev/null 2>&1; then
    echo -e "  Backend ($BACKEND_URL): ${GREEN}OK${NC}"
else
    echo -e "  Backend ($BACKEND_URL): ${YELLOW}NOT REACHABLE${NC}"
    echo -e "${YELLOW}Attempting to start backend...${NC}"

    # Try to start backend
    BACKEND_DIR="$(cd "$SCRIPT_DIR/../../backend" && pwd)"
    if [ -f "$BACKEND_DIR/main.go" ]; then
        echo -e "  Starting backend from $BACKEND_DIR..."
        cd "$BACKEND_DIR"
        source .env 2>/dev/null || true
        go run . &
        BACKEND_PID=$!
        cd "$SCRIPT_DIR"

        echo -e "  Waiting for backend to start..."
        for i in $(seq 1 30); do
            if curl -s "$BACKEND_URL/api/v1/agents" >/dev/null 2>&1; then
                echo -e "  Backend started: ${GREEN}OK${NC} (PID: $BACKEND_PID)"
                break
            fi
            sleep 1
        done

        if ! curl -s "$BACKEND_URL/api/v1/agents" >/dev/null 2>&1; then
            echo -e "  ${RED}Backend failed to start${NC}"
            kill $BACKEND_PID 2>/dev/null || true
            exit 1
        fi
    else
        echo -e "${RED}Cannot find backend. Please start it manually.${NC}"
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}Running E2E tests...${NC}"
echo ""

# Create reports directory
mkdir -p reports/screenshots

# Run pytest with HTML report
python3 -m pytest \
    --html=reports/report.html \
    --self-contained-html \
    --tb=short \
    --timeout=120 \
    -v \
    "$@" 2>&1 | tee reports/test_output.log

EXIT_CODE=${PIPESTATUS[0]}

# Generate markdown summary
python3 generate_report.py

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  ALL TESTS PASSED${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}  SOME TESTS FAILED (exit: $EXIT_CODE)${NC}"
    echo -e "${RED}========================================${NC}"
fi

echo ""
echo -e "Reports:"
echo -e "  HTML: ${SCRIPT_DIR}/reports/report.html"
echo -e "  Markdown: ${SCRIPT_DIR}/reports/report.md"
echo -e "  Screenshots: ${SCRIPT_DIR}/reports/screenshots/"
echo ""

# Kill backend if we started it
if [ -n "${BACKEND_PID:-}" ]; then
    echo -e "${YELLOW}Stopping backend (PID: $BACKEND_PID)...${NC}"
    kill $BACKEND_PID 2>/dev/null || true
fi

exit $EXIT_CODE
