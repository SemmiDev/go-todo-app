#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  backup-db.sh — PostgreSQL Database Backup                  ║
# ║                                                              ║
# ║  Usage:                                                      ║
# ║    ./scripts/backup-db.sh                  # Default backup  ║
# ║    ./scripts/backup-db.sh /custom/path     # Custom path     ║
# ╚══════════════════════════════════════════════════════════════╝

set -euo pipefail

# ── Configuration ────────────────────────────────────────────
CONTAINER_NAME="gotodoapp-postgres"
DB_USER="${POSTGRES_USER:-gotodoapp}"
DB_NAME="${POSTGRES_DB:-gotodoapp}"
BACKUP_DIR="${1:-/opt/go-todo-app/backups}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/gotodoapp_${TIMESTAMP}.sql.gz"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
TELEGRAM_CHAT_ID="${TELEGRAM_CHAT_ID:-}"

# ── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}📦 Starting PostgreSQL backup...${NC}"

# ── Pre-flight ───────────────────────────────────────────────
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${RED}❌ Container '${CONTAINER_NAME}' is not running!${NC}"
    exit 1
fi

mkdir -p "${BACKUP_DIR}"

# ── Backup ───────────────────────────────────────────────────
echo "   Database: ${DB_NAME}"
echo "   Output:   ${BACKUP_FILE}"

docker exec "${CONTAINER_NAME}" \
    pg_dump -U "${DB_USER}" -d "${DB_NAME}" \
    --verbose --clean --if-exists --create \
    2>/dev/null | gzip > "${BACKUP_FILE}"

# ── Verify ───────────────────────────────────────────────────
BACKUP_SIZE=$(ls -lh "${BACKUP_FILE}" | awk '{print $5}')
echo -e "${GREEN}✅ Backup complete: ${BACKUP_FILE} (${BACKUP_SIZE})${NC}"

# ── Send to Telegram ─────────────────────────────────────────
if [[ -n "${TELEGRAM_BOT_TOKEN}" && -n "${TELEGRAM_CHAT_ID}" ]]; then
    echo -e "${YELLOW}📤 Sending backup to Telegram...${NC}"
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendDocument" \
        -F chat_id="${TELEGRAM_CHAT_ID}" \
        -F document=@"${BACKUP_FILE}" \
        -F caption="✅ gotodoapp-postgres Database Backup (${BACKUP_SIZE})" || echo "")

    HTTP_CODE=$(echo "${RESPONSE}" | tail -n1)
    BODY=$(echo "${RESPONSE}" | sed '$d')

    if [[ "${HTTP_CODE}" == "200" ]]; then
        echo -e "${GREEN}✅ Backup successfully sent to Telegram${NC}"
    else
        echo -e "${RED}❌ Failed to send backup to Telegram (HTTP ${HTTP_CODE}):${NC}"
        echo "${BODY}"
    fi
elif [[ -n "${TELEGRAM_BOT_TOKEN}" && -z "${TELEGRAM_CHAT_ID}" ]]; then
    echo -e "${YELLOW}⚠️ TELEGRAM_CHAT_ID not set. Skipping Telegram notification.${NC}"
    echo -e "${YELLOW}   Set TELEGRAM_CHAT_ID environment variable to receive backups.${NC}"
fi

# ── Cleanup Old Backups ─────────────────────────────────────
DELETED=$(find "${BACKUP_DIR}" -name "gotodoapp_*.sql.gz" -mtime +${RETENTION_DAYS} -delete -print | wc -l)
if [[ ${DELETED} -gt 0 ]]; then
    echo -e "${YELLOW}🧹 Removed ${DELETED} backup(s) older than ${RETENTION_DAYS} days${NC}"
fi

# ── List Recent Backups ──────────────────────────────────────
echo ""
echo "Recent backups:"
ls -lhrt "${BACKUP_DIR}"/gotodoapp_*.sql.gz 2>/dev/null | tail -5
