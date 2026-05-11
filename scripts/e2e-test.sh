#!/bin/bash
set -e

BASE_URL="${1:-http://localhost:8888}"
GRPC_ADDR="${2:-localhost:50051}"

echo "=== Superagent Base E2E Tests ==="
echo "Backend: $BASE_URL"
echo ""

PASS=0
FAIL=0

test_endpoint() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    result=$(eval "$cmd" 2>&1)
    if echo "$result" | grep -q "$expected"; then
        echo "  ✅ $name"
        PASS=$((PASS + 1))
    else
        echo "  ❌ $name"
        echo "     Expected: $expected"
        echo "     Got: $(echo $result | head -c 200)"
        FAIL=$((FAIL + 1))
    fi
}

echo "[Health Checks]"
test_endpoint "HTTP server responds" \
    "curl -s -o /dev/null -w '%{http_code}' $BASE_URL/" "200\|404"

test_endpoint "Metrics endpoint" \
    "curl -s $BASE_URL/metrics" "superagent_"

echo ""
echo "[Model Connectivity]"
test_endpoint "LLM responds" \
    "curl -s -X POST http://127.0.0.1:8000/v1/chat/completions -H 'Content-Type: application/json' -H 'Authorization: Bearer 123456' -d '{\"model\":\"Qwen3-Coder-Next-4bit\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"max_tokens\":5}'" "choices"

echo ""
echo "=============================="
echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && echo "All tests passed!" || exit 1
