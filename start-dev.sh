#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}Starting Workspace Development Environment...${NC}"

# 1. Cleanup conflicting ports/containers
echo -e "${YELLOW}Checking for conflicting containers...${NC}"
if docker ps | grep -q "nucleus-temporal-1"; then
    echo "Stopping nucleus-temporal-1 to free port 7233..."
    docker stop nucleus-temporal-1
fi

if docker ps | grep -q "workspace-"; then
    echo "Stopping existing workspace containers..."
    docker compose down
fi

# 2. Start Infrastructure (Docker)
echo -e "${BLUE}Starting Infrastructure (Postgres, Temporal, Keystore, MCP)...${NC}"
docker compose up -d postgres temporal keystore mcp-server

# Wait for healthy services
echo "Waiting for Postgres..."
for i in {1..30}; do
    if docker compose ps postgres | grep -q "healthy"; then
        echo -e "${GREEN}Postgres is ready!${NC}"
        break
    fi
    sleep 1
done

echo "Waiting for Temporal..."
# Simple sleep as Temporal healthcheck isn't always exposed in ps
sleep 5

# 3. Start Local Services
# Set Environment Variables for Local Dev
export POSTGRES_URL="postgres://agent:devpassword@localhost:5442/agent?sslmode=disable"
export TEMPORAL_HOST="localhost:7233"
export NUCLEUS_URL="${NUCLEUS_URL:-http://localhost:4000}"
export NUCLEUS_API_URL="${NUCLEUS_API_URL:-http://localhost:4000/graphql}"
export NUCLEUS_UCL_URL="${NUCLEUS_UCL_URL:-localhost:50051}"
export GRPC_PORT="9000"
export GEMINI_API_KEY="${GEMINI_API_KEY}" # Inherit (ensure set in shell)
export MCP_SERVER_URL="http://localhost:9100"

# Gateway Config
export GATEWAY_PORT="8080"
export AGENT_SERVICE_URL="http://localhost:9000"

# Kill running instances if any
pkill -f "go-agent-service" || true
pkill -f "rust-gateway" || true
pkill -f "next dev" || true

echo -e "${BLUE}Starting Services...${NC}"

# Go Agent Service
echo "Starting Go Agent Service..."
cd go-agent-service
go run ./cmd/server > ../agent.log 2>&1 &
AGENT_PID=$!
echo -e "${GREEN}Go Agent running (PID: $AGENT_PID)${NC}"
cd ..

# Rust Gateway
echo "Starting Rust Gateway..."
cd rust-gateway
# checks if cargo is available
if command -v cargo &> /dev/null; then
    cargo run > ../gateway.log 2>&1 &
    GATEWAY_PID=$!
    echo -e "${GREEN}Rust Gateway running (PID: $GATEWAY_PID)${NC}"
else
    echo -e "${YELLOW}Cargo not found, skipping Rust Gateway local run.${NC}"
fi
cd ..

# Web Frontend
echo "Starting Workspace Web..."
cd workspace-web
npm run dev > ../web.log 2>&1 &
WEB_PID=$!
echo -e "${GREEN}Workspace Web running (PID: $WEB_PID)${NC}"
cd ..

echo -e "${BLUE}Environment is UP!${NC}"
echo -e "  - Web: http://localhost:3000"
echo -e "  - Gateway: http://localhost:8080"
echo -e "  - Agent: http://localhost:9000"
echo -e "  - MCP: http://localhost:9100"
echo -e "  - Keystore: http://localhost:9200"
echo -e "  - Temporal UI: http://localhost:8233"
echo -e "  - Nucleus: ${NUCLEUS_API_URL}"
echo ""
echo "Logs are being written to agent.log, gateway.log, and web.log"
echo "Press Ctrl+C to stop all services."

# Trap SIGINT to kill background processes
trap "kill $AGENT_PID $GATEWAY_PID $WEB_PID; docker compose stop; exit" INT

# Wait structure to keep script running
wait
