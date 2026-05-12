#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REGISTRY="${LOCAL_REGISTRY:-localhost:5001}"
NS="superagent"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== Superagent K8s E2E Deployment ===${NC}"

# Step 1: Build image
echo -e "${YELLOW}[1/5] Building Linux binary...${NC}"
cd "$PROJECT_ROOT/backend"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o /tmp/superagent-linux .

echo -e "${YELLOW}[2/5] Building Docker image...${NC}"
cd /tmp
cp -r "$PROJECT_ROOT/backend/conf" . 2>/dev/null || true
cp -r "$PROJECT_ROOT/backend/configs/agents" . 2>/dev/null || true

cat > Dockerfile.k8s <<'EOF'
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY superagent-linux ./superagent
COPY conf ./resources/conf
COPY agents ./configs/agents
RUN chmod +x ./superagent
EXPOSE 8888 50051
ENTRYPOINT ["/app/superagent"]
EOF

docker build -f Dockerfile.k8s -t superagent-base:latest .
docker tag superagent-base:latest "$REGISTRY/superagent-base:latest"

# Step 3: Load into kind nodes
echo -e "${YELLOW}[3/5] Loading image into kind nodes...${NC}"
for node in $(kubectl get nodes -o name | sed 's|node/||'); do
  echo "  Loading into $node..."
  docker save superagent-base:latest | docker exec -i "$node" ctr -n k8s.io images import - 2>/dev/null || true
done

# Step 4: Apply manifests
echo -e "${YELLOW}[4/5] Applying K8s manifests...${NC}"
cd "$SCRIPT_DIR"
kubectl apply -f namespace.yaml
kubectl apply -f middleware.yaml
kubectl apply -f superagent.yaml
kubectl apply -f monitoring.yaml

# Patch to use local image
kubectl patch deployment superagent -n $NS \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"superagent","image":"docker.io/library/superagent-base:latest","imagePullPolicy":"Never"}]}}}}'

# Step 5: Wait and verify
echo -e "${YELLOW}[5/5] Waiting for pods...${NC}"
kubectl wait --for=condition=ready pod -l app=mysql -n $NS --timeout=120s 2>/dev/null || true
kubectl wait --for=condition=ready pod -l app=redis -n $NS --timeout=60s 2>/dev/null || true
kubectl wait --for=condition=ready pod -l app=superagent -n $NS --timeout=120s 2>/dev/null || true

echo ""
echo -e "${GREEN}=== Deployment Status ===${NC}"
kubectl get pods -n $NS -o wide
echo ""
echo -e "${GREEN}=== Services ===${NC}"
kubectl get svc -n $NS
echo ""

# Test connectivity
SA_IP=$(kubectl get pod -n $NS -l app=superagent -o jsonpath='{.items[0].status.podIP}' 2>/dev/null || echo "")
if [ -n "$SA_IP" ]; then
  echo -e "${GREEN}=== Health Check ===${NC}"
  kubectl exec -n $NS deployment/superagent -- wget -qO- http://localhost:8888/health 2>/dev/null || echo "Health check pending..."
fi

echo ""
echo -e "${GREEN}Access Points:${NC}"
echo "  Superagent API: kubectl port-forward svc/superagent -n $NS 8888:8888"
echo "  Prometheus UI:  kubectl port-forward svc/prometheus -n $NS 9090:9090"
echo "  Grafana UI:     kubectl port-forward svc/grafana -n $NS 3000:3000"
echo ""
echo -e "  Or via NodePort:"
echo "  Superagent: http://localhost:30888"
echo "  Prometheus: http://localhost:30190"
echo "  Grafana:    http://localhost:30030 (admin/admin)"
