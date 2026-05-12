#!/usr/bin/env bash
set -euo pipefail

NS="superagent"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== K8s Deployment Verification ===${NC}"
echo ""

# 1. Pod Status
echo -e "${YELLOW}[1] Pod Status:${NC}"
kubectl get pods -n $NS
echo ""

# 2. Service Status
echo -e "${YELLOW}[2] Services:${NC}"
kubectl get svc -n $NS
echo ""

# 3. Health Check
echo -e "${YELLOW}[3] Superagent Health:${NC}"
SA_POD=$(kubectl get pod -n $NS -l app=superagent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$SA_POD" ]; then
  HEALTH=$(kubectl exec -n $NS "$SA_POD" -- wget -qO- http://localhost:8888/health 2>/dev/null || echo "FAIL")
  READY=$(kubectl exec -n $NS "$SA_POD" -- wget -qO- http://localhost:8888/ready 2>/dev/null || echo "FAIL")
  AGENTS=$(kubectl exec -n $NS "$SA_POD" -- wget -qO- http://localhost:8888/api/v1/agents 2>/dev/null || echo "FAIL")
  echo "  /health: $HEALTH"
  echo "  /ready:  $READY"
  echo "  /agents: $AGENTS"
else
  echo -e "  ${RED}No superagent pod running${NC}"
fi
echo ""

# 4. Metrics
echo -e "${YELLOW}[4] Prometheus Metrics:${NC}"
PROM_POD=$(kubectl get pod -n $NS -l app=prometheus -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$PROM_POD" ]; then
  TARGETS=$(kubectl exec -n $NS "$PROM_POD" -- wget -qO- 'http://localhost:9090/api/v1/targets' 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); active=[t for t in d.get('data',{}).get('activeTargets',[])]; print(f'  Active targets: {len(active)}'); [print(f'    - {t[\"labels\"].get(\"job\",\"?\")} ({t[\"health\"]})') for t in active]" 2>/dev/null || echo "  Prometheus query failed")
  echo "$TARGETS"
else
  echo -e "  ${RED}No prometheus pod running${NC}"
fi
echo ""

# 5. Chat Test
echo -e "${YELLOW}[5] Chat Test:${NC}"
if [ -n "$SA_POD" ]; then
  CHAT=$(kubectl exec -n $NS "$SA_POD" -- wget -qO- --post-data='{"agent_id":"research-agent","session_id":"k8s-test","message":"say hello"}' --header='Content-Type: application/json' http://localhost:8888/api/v1/chat/stream 2>/dev/null || echo "FAIL")
  echo "  Response: ${CHAT:0:200}"
else
  echo -e "  ${RED}Skipped (no pod)${NC}"
fi
echo ""

# Summary
echo -e "${GREEN}=== Summary ===${NC}"
TOTAL=$(kubectl get pods -n $NS --no-headers 2>/dev/null | wc -l)
RUNNING=$(kubectl get pods -n $NS --no-headers 2>/dev/null | grep -c "Running" || echo "0")
echo -e "  Pods: $RUNNING/$TOTAL Running"

if [ "$RUNNING" -eq "$TOTAL" ] && [ "$TOTAL" -gt 0 ]; then
  echo -e "  ${GREEN}ALL PODS HEALTHY${NC}"
else
  echo -e "  ${YELLOW}Some pods not ready — check status above${NC}"
fi
