#!/bin/bash
cd "$(dirname "$0")/.."
echo "Stopping backend..."
pkill -f "bin/superagent" 2>/dev/null || true
echo "Stopping containers..."
docker compose -f docker/docker-compose-dev.yml down
echo "Done."
