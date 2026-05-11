#!/bin/bash
set -e

echo "=== Superagent Base - Dev Environment ==="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Docker is required"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Go is required"; exit 1; }

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

# Step 1: Start middleware
echo -e "${GREEN}[1/4] Starting MySQL + Redis...${NC}"
docker compose -f docker/docker-compose-dev.yml up -d --wait

# Step 2: Build backend
echo -e "${GREEN}[2/4] Building backend...${NC}"
cd backend
go build -o ../bin/superagent .
cd ..

# Step 3: Start backend
echo -e "${GREEN}[3/4] Starting backend server...${NC}"
export APP_ENV=dev
cp docker/.env.dev backend/.env.dev 2>/dev/null || true
cd backend && APP_ENV=dev ../bin/superagent &
BACKEND_PID=$!
cd ..

# Wait for backend
sleep 3
echo -e "${GREEN}[4/4] Backend started (PID: $BACKEND_PID)${NC}"

echo ""
echo "=============================="
echo -e "${GREEN}Superagent Base is running!${NC}"
echo ""
echo "  Backend:  http://localhost:8888"
echo "  gRPC:     localhost:50051"
echo "  Metrics:  http://localhost:8888/metrics"
echo ""
echo "  MySQL:    localhost:3306"
echo "  Redis:    localhost:6379"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo "=============================="

# Trap and cleanup
trap "kill $BACKEND_PID 2>/dev/null; docker compose -f docker/docker-compose-dev.yml down; exit" INT TERM
wait $BACKEND_PID
