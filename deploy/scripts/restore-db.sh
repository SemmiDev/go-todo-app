#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  restore-db.sh — PostgreSQL Database Restore                ║
# ║                                                              ║
# ║  Usage:                                                      ║
# ║    ./scripts/restore-db.sh <backup_file.sql.gz>              ║
# ║    ./scripts/restore-db.sh /opt/go-todo-app/backups/gotodoapp_20260223.sql.gz
# ╚══════════════════════════════════════════════════════════════╝

set -euo pipefail

# ── Configuration ────────────────────────────────────────────
CONTAINER_NAME="gotodoapp-postgres"
DB_USER="${POSTGRES_USER:-gotodoapp}"
DB_NAME="${POSTGRES_DB:-gotodoapp}"

# ── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ── Validate Input ──────────────────────────────────────────
if [[ $# -lt 1 ]]; then
    echo -e "${RED}❌ Usage: $0 <backup_file.sql.gz>${NC}"
    echo ""
    echo "Available backups:"
    ls -lhrt /opt/go-todo-app/backups/gotodoapp_*.sql.gz 2>/dev/null | tail -10
    exit 1
fi

BACKUP_FILE="$1"

if [[ ! -f "${BACKUP_FILE}" ]]; then
    echo -e "${RED}❌ Backup file not found: ${BACKUP_FILE}${NC}"
    exit 1
fi

# ── Pre-flight ───────────────────────────────────────────────
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${RED}❌ Container '${CONTAINER_NAME}' is not running!${NC}"
    exit 1
fi

BACKUP_SIZE=$(ls -lh "${BACKUP_FILE}" | awk '{print $5}')
echo -e "${YELLOW}⚠️  WARNING: This will REPLACE the current database!${NC}"
echo ""
echo "   Database:    ${DB_NAME}"
echo "   Backup file: ${BACKUP_FILE} (${BACKUP_SIZE})"
echo ""

read -p "$(echo -e ${RED}"▶ Are you SURE? Type 'yes' to proceed: "${NC})" confirm
if [[ "$confirm" != "yes" ]]; then
    echo -e "${YELLOW}❌ Restore cancelled.${NC}"
    exit 0
fi

# ── Create safety backup first ──────────────────────────────
echo ""
echo -e "${YELLOW}📦 Creating safety backup before restore...${NC}"
SAFETY_BACKUP="/opt/go-todo-app/backups/pre_restore_$(date +%Y%m%d_%H%M%S).sql.gz"

docker exec "${CONTAINER_NAME}" \
    pg_dump -U "${DB_USER}" -d "${DB_NAME}" \
    --clean --if-exists 2>/dev/null | gzip > "${SAFETY_BACKUP}"

echo -e "${GREEN}   Safety backup: ${SAFETY_BACKUP}${NC}"

# ── Restore ──────────────────────────────────────────────────
echo ""
echo -e "${YELLOW}🔄 Restoring database...${NC}"

gunzip -c "${BACKUP_FILE}" | docker exec -i "${CONTAINER_NAME}" \
    psql -U "${DB_USER}" -d "${DB_NAME}" \
    --quiet --single-transaction 2>/dev/null

echo ""
echo -e "${GREEN}✅ Database restored successfully from: ${BACKUP_FILE}${NC}"
echo -e "${GREEN}   Safety backup available at: ${SAFETY_BACKUP}${NC}"
