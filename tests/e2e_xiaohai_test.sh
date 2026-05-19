#!/bin/bash
# E2E Test: 集团IT智能体输出规范接口 + 现有接口兼容性
# =============================================================================

BASE_URL="http://localhost:8888"
PASS=0
FAIL=0
RESULTS=""

check() {
  local name="$1" cond="$2"
  if eval "$cond"; then
    PASS=$((PASS+1))
    RESULTS="${RESULTS}\n✅ PASS: ${name}"
  else
    FAIL=$((FAIL+1))
    RESULTS="${RESULTS}\n❌ FAIL: ${name}"
  fi
}

echo "=========================================="
echo "  E2E Test Suite: Superagent Base"
echo "  Date: $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="

# ─── 1. Health & Ready ───────────────────────────────────────────────────────
echo -e "\n[1/7] Health & Ready checks..."

HEALTH=$(curl -s $BASE_URL/health)
check "GET /health returns ok" '[ "$(echo $HEALTH | python3 -c "import sys,json;print(json.load(sys.stdin)[\"status\"])")" = "ok" ]'

READY=$(curl -s $BASE_URL/ready)
check "GET /ready returns 200" '[ "$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/ready)" = "200" ]'

# ─── 2. Agent List ───────────────────────────────────────────────────────────
echo -e "\n[2/7] Agent listing..."

AGENTS=$(curl -s $BASE_URL/api/v1/agents)
AGENT_COUNT=$(echo $AGENTS | python3 -c "import sys,json;print(len(json.load(sys.stdin).get('agents',[])))")
check "GET /api/v1/agents returns agents" '[ "$AGENT_COUNT" -gt 0 ]'
check "research-agent exists" 'echo $AGENTS | grep -q "research-agent"'
check "approval-agent exists" 'echo $AGENTS | grep -q "approval-agent"'

# ─── 3. 现有 /api/v1/chat/stream (Legacy mode) ─────────────────────────────
echo -e "\n[3/7] Legacy SSE streaming..."

LEGACY=$(curl -s -X POST $BASE_URL/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"research-agent","session_id":"e2e-legacy-1","message":"hello"}' \
  --max-time 30 2>&1)
check "Legacy SSE returns event: message" 'echo "$LEGACY" | grep -q "event: message"'
check "Legacy SSE returns data lines" 'echo "$LEGACY" | grep -q "^data:"'

# ─── 4. 现有 /api/v1/chat/stream (A2UI mode) ────────────────────────────────
echo -e "\n[4/7] A2UI SSE streaming..."

A2UI=$(curl -s -X POST $BASE_URL/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -H "X-A2UI: true" \
  -d '{"agent_id":"research-agent","session_id":"e2e-a2ui-1","message":"hi"}' \
  --max-time 30 2>&1)
check "A2UI SSE returns event: text" 'echo "$A2UI" | grep -q "event: text"'
check "A2UI SSE data contains type field" 'echo "$A2UI" | grep "^data:" | head -1 | grep -q "\"type\""'
check "A2UI SSE data contains delta field" 'echo "$A2UI" | grep "^data:" | head -1 | grep -q "\"delta\""'

# ─── 5. 小海流式接口 /api/v1/xiaohai/stream/:agent_id ───────────────────────
echo -e "\n[5/7] 小海流式接口 (集团规范)..."

XH_STREAM=$(curl -s -X POST $BASE_URL/api/v1/xiaohai/stream/research-agent \
  -H "Content-Type: application/json" \
  -d '{"userQuery":"你好","sessionId":"e2e-xh-1","terminalType":"PC"}' \
  --max-time 30 2>&1)
check "Xiaohai stream returns data lines" 'echo "$XH_STREAM" | grep -q "^data:"'
check "Xiaohai stream has type=answer" 'echo "$XH_STREAM" | grep -q "\"type\":\"answer\""'
check "Xiaohai stream has content_type=markdown" 'echo "$XH_STREAM" | grep -q "\"content_type\":\"markdown\""'
check "Xiaohai stream has version=1.0.0" 'echo "$XH_STREAM" | grep -q "\"version\":\"1.0.0\""'
check "Xiaohai stream ends with stream_end" 'echo "$XH_STREAM" | grep -q "\"type\":\"stream_end\""'

# Tool call test
XH_TOOL=$(curl -s -X POST $BASE_URL/api/v1/xiaohai/stream/approval-agent \
  -H "Content-Type: application/json" \
  -d '{"userQuery":"帮我请求 https://httpbin.org/ip","sessionId":"e2e-xh-tool","terminalType":"PC"}' \
  --max-time 60 2>&1)
check "Xiaohai tool call emits execution_steps" 'echo "$XH_TOOL" | grep -q "\"type\":\"execution_steps\""'
check "Xiaohai tool call emits execution_steps_end" 'echo "$XH_TOOL" | grep -q "\"type\":\"execution_steps_end\""'

# ─── 6. 小海非流式接口 /api/v1/xiaohai/chat/:agent_id ───────────────────────
echo -e "\n[6/7] 小海非流式接口..."

XH_CHAT=$(curl -s -X POST $BASE_URL/api/v1/xiaohai/chat/research-agent \
  -H "Content-Type: application/json" \
  -d '{"userQuery":"1+1","sessionId":"e2e-xh-chat","terminalType":"PC"}' \
  --max-time 30 2>&1)
XH_CODE=$(echo $XH_CHAT | python3 -c "import sys,json;print(json.load(sys.stdin).get('code',''))" 2>/dev/null)
check "Xiaohai chat returns code=0" '[ "$XH_CODE" = "0" ]'
check "Xiaohai chat data has type=answer" 'echo "$XH_CHAT" | grep -q "\"type\":\"answer\""'
check "Xiaohai chat data has content_type=markdown" 'echo "$XH_CHAT" | grep -q "\"content_type\":\"markdown\""'
check "Xiaohai chat data has version=1.0.0" 'echo "$XH_CHAT" | grep -q "\"version\":\"1.0.0\""'
check "Xiaohai chat content is non-empty" 'echo "$XH_CHAT" | python3 -c "import sys,json;d=json.load(sys.stdin);print(len(str(d.get(\"data\",{}).get(\"data\",{}).get(\"content\",\"\")))>0)" 2>/dev/null | grep -q "True"'

# ─── 7. 错误处理 ────────────────────────────────────────────────────────────
echo -e "\n[7/7] Error handling..."

ERR_404=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE_URL/api/v1/xiaohai/stream/nonexistent-agent \
  -H "Content-Type: application/json" \
  -d '{"userQuery":"test","sessionId":"e2e-err","terminalType":"PC"}')
check "Xiaohai 404 for unknown agent" '[ "$ERR_404" = "404" ]'

ERR_400=$(curl -s -o /dev/null -w "%{http_code}" -X POST $BASE_URL/api/v1/xiaohai/stream/research-agent \
  -H "Content-Type: application/json" \
  -d '{"userQuery":"","sessionId":"e2e-err","terminalType":"PC"}')
check "Xiaohai 400 for empty userQuery" '[ "$ERR_400" = "400" ]'

# ─── Report ──────────────────────────────────────────────────────────────────
echo -e "\n=========================================="
echo "  TEST RESULTS"
echo "=========================================="
echo -e "$RESULTS"
echo ""
echo "Total: $((PASS+FAIL)) | Pass: $PASS | Fail: $FAIL"
echo "=========================================="

if [ $FAIL -gt 0 ]; then
  exit 1
fi
