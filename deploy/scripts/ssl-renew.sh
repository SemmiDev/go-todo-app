#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  ssl-renew.sh — Let's Encrypt SSL Certificate Renewal       ║
# ║                                                              ║
# ║  Usage:                                                      ║
# ║    ./scripts/ssl-renew.sh                                    ║
# ║    ./scripts/ssl-renew.sh --force                            ║
# ║                                                              ║
# ║  Cron (auto-renewal every day at 2 AM):                      ║
# ║    0 2 * * * /opt/go-todo-app/scripts/ssl-renew.sh            ║
# ╚══════════════════════════════════════════════════════════════╝

set -euo pipefail

# ── Configuration ────────────────────────────────────────────
LOG_FILE="/var/log/gotodoapp-ssl-renew.log"
FORCE="${1:-}"

# ── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "${LOG_FILE}"
}

log "Starting SSL certificate renewal check..."

# ── Check Certbot ────────────────────────────────────────────
if ! command -v certbot &>/dev/null; then
    log "ERROR: certbot not installed"
    exit 1
fi

# ── Renew ────────────────────────────────────────────────────
CERTBOT_ARGS="renew --quiet --post-hook 'systemctl reload nginx'"

if [[ "${FORCE}" == "--force" ]]; then
    CERTBOT_ARGS="renew --force-renewal --post-hook 'systemctl reload nginx'"
    log "Force renewal requested"
fi

if eval certbot ${CERTBOT_ARGS} 2>&1 | tee -a "${LOG_FILE}"; then
    log "SSL renewal check completed successfully"
else
    log "ERROR: SSL renewal failed!"
    exit 1
fi

# ── Display Certificate Info ─────────────────────────────────
echo ""
echo -e "${YELLOW}Current certificates:${NC}"
certbot certificates 2>/dev/null | grep -E "(Certificate Name|Expiry Date|Domains)" || true

echo ""
echo -e "${GREEN}✅ SSL renewal check complete${NC}"
