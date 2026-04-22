#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  health-check.sh — Application Health Check                  ║
# ║                                                              ║
# ║  Usage:                                                      ║
# ║    ./scripts/health-check.sh                                 ║
# ║    ./scripts/health-check.sh --verbose                       ║
# ╚══════════════════════════════════════════════════════════════╝

set -euo pipefail

# ── Configuration ────────────────────────────────────────────
APP_URL="${APP_URL:-http://127.0.0.1:8080}"
CONTAINER_NAME="${CONTAINER_NAME:-todo-app}"
DB_CONTAINER="${DB_CONTAINER:-gotodoapp-postgres}"
VERBOSE="${1:-}"

# ── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

ERRORS=0

echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  todo-app — Health Check                   ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""

# ── Check 1: Docker daemon ──────────────────────────────────
echo -n "1. Docker daemon .............. "
if docker info &>/dev/null; then
    echo -e "${GREEN}✅ Running${NC}"
else
    echo -e "${RED}❌ Not running${NC}"
    ((ERRORS++))
fi

# ── Check 2: Application container ──────────────────────────
echo -n "2. App container .............. "
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    STATUS=$(docker inspect --format='{{.State.Status}}' "${CONTAINER_NAME}")
    echo -e "${GREEN}✅ ${STATUS}${NC}"
else
    echo -e "${RED}❌ Not running${NC}"
    ((ERRORS++))
fi

# ── Check 3: PostgreSQL container ───────────────────────────
echo -n "3. PostgreSQL container ....... "
if docker ps --format '{{.Names}}' | grep -q "^${DB_CONTAINER}$"; then
    echo -e "${GREEN}✅ Running${NC}"
else
    echo -e "${RED}❌ Not running${NC}"
    ((ERRORS++))
fi

# ── Check 4: Database connectivity ──────────────────────────
echo -n "4. Database connectivity ...... "
if docker exec "${DB_CONTAINER}" pg_isready -U gotodoapp -d gotodoapp &>/dev/null; then
    echo -e "${GREEN}✅ Ready${NC}"
else
    echo -e "${RED}❌ Not ready${NC}"
    ((ERRORS++))
fi

# ── Check 5: HTTP response ──────────────────────────────────
echo -n "5. HTTP response .............. "
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "${APP_URL}" 2>/dev/null || echo "000")
if [[ "${HTTP_CODE}" =~ ^(200|301|302)$ ]]; then
    echo -e "${GREEN}✅ HTTP ${HTTP_CODE}${NC}"
else
    echo -e "${RED}❌ HTTP ${HTTP_CODE}${NC}"
    ((ERRORS++))
fi

# ── Check 6: Nginx ──────────────────────────────────────────
echo -n "6. Nginx ...................... "
if systemctl is-active --quiet nginx 2>/dev/null; then
    echo -e "${GREEN}✅ Running${NC}"
else
    echo -e "${YELLOW}⚠️  Not running (may not be installed)${NC}"
fi

# ── Check 7: Disk Usage ─────────────────────────────────────
echo -n "7. Disk usage ................. "
DISK_USAGE=$(df -h / | awk 'NR==2 {print $5}' | tr -d '%')
if [[ ${DISK_USAGE} -lt 80 ]]; then
    echo -e "${GREEN}✅ ${DISK_USAGE}% used${NC}"
elif [[ ${DISK_USAGE} -lt 90 ]]; then
    echo -e "${YELLOW}⚠️  ${DISK_USAGE}% used${NC}"
else
    echo -e "${RED}❌ ${DISK_USAGE}% used — CRITICAL${NC}"
    ((ERRORS++))
fi

# ── Check 8: Memory ─────────────────────────────────────────
echo -n "8. Memory usage ............... "
MEM_USAGE=$(free | awk '/^Mem:/ {printf "%d", $3/$2 * 100}')
if [[ ${MEM_USAGE} -lt 80 ]]; then
    echo -e "${GREEN}✅ ${MEM_USAGE}% used${NC}"
elif [[ ${MEM_USAGE} -lt 90 ]]; then
    echo -e "${YELLOW}⚠️  ${MEM_USAGE}% used${NC}"
else
    echo -e "${RED}❌ ${MEM_USAGE}% used — CRITICAL${NC}"
    ((ERRORS++))
fi

# ── Verbose Details ──────────────────────────────────────────
if [[ "${VERBOSE}" == "--verbose" || "${VERBOSE}" == "-v" ]]; then
    echo ""
    echo -e "${BLUE}── Container Details ──${NC}"
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.Size}}" \
        --filter "name=gotodoapp" 2>/dev/null

    echo ""
    echo -e "${BLUE}── Recent App Logs ──${NC}"
    docker logs "${CONTAINER_NAME}" --tail 10 2>/dev/null || echo "  No logs available"

    echo ""
    echo -e "${BLUE}── Docker Images ──${NC}"
    docker images todo-app --format "table {{.Tag}}\t{{.CreatedAt}}\t{{.Size}}" 2>/dev/null
fi

# ── Summary ──────────────────────────────────────────────────
echo ""
if [[ ${ERRORS} -eq 0 ]]; then
    echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ✅ All checks passed!                       ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔══════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║  ❌ ${ERRORS} check(s) failed!                      ║${NC}"
    echo -e "${RED}╚══════════════════════════════════════════════╝${NC}"
    exit 1
fi
