#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  deploy.sh — One-command deployment wrapper                  ║
# ║                                                              ║
# ║  Usage:                                                      ║
# ║    ./scripts/deploy.sh                    # Full deploy      ║
# ║    ./scripts/deploy.sh --setup            # Initial setup    ║
# ║    ./scripts/deploy.sh --app-only         # App only         ║
# ║    ./scripts/deploy.sh --staging          # Staging env      ║
# ║    ./scripts/deploy.sh --env NAME         # Custom env       ║
# ║    ./scripts/deploy.sh --branch dev       # Custom branch    ║
# ╚══════════════════════════════════════════════════════════════╝

set -euo pipefail

# ── Colors ───────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# ── Defaults ─────────────────────────────────────────────────
INVENTORY="inventory/production.ini"
PLAYBOOK="playbooks/site.yml"
EXTRA_VARS=()
TAGS=()

# ── Parse Arguments ──────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case $1 in
        --setup)
            PLAYBOOK="playbooks/setup.yml"
            shift ;;
        --app-only)
            PLAYBOOK="playbooks/deploy.yml"
            shift ;;
        --staging)
            INVENTORY="inventory/staging.ini"
            EXTRA_VARS+=("-e" "@group_vars/staging.yml")
            shift ;;
        --env)
            EXTRA_VARS+=("-e" "env=$2")
            shift 2 ;;
        --branch)
            EXTRA_VARS+=("-e" "app_branch=$2")
            shift 2 ;;
        --tags)
            TAGS+=("--tags" "$2")
            shift 2 ;;
        --dry-run)
            EXTRA_VARS+=("--check" "--diff")
            shift ;;
        --verbose)
            EXTRA_VARS+=("-vvv")
            shift ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --setup       Run initial server setup only"
            echo "  --app-only    Deploy/update application only"
            echo "  --staging     Use staging environment"
            echo "  --env NAME    Use specified environment variables"
            echo "  --branch NAME Deploy specific branch (default: main)"
            echo "  --tags TAGS   Run only specific Ansible tags"
            echo "  --dry-run     Preview changes without applying"
            echo "  --verbose     Enable verbose Ansible output"
            echo "  --help        Show this help message"
            exit 0 ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1 ;;
    esac
done

# ── Pre-flight Checks ───────────────────────────────────────
echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  go-todo-app Deployment                       ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""

# Check Ansible is installed
if ! command -v ansible-playbook &>/dev/null; then
    echo -e "${RED}❌ ansible-playbook not found. Install with:${NC}"
    echo "   pip install ansible"
    exit 1
fi

# Check inventory file exists
if [[ ! -f "${DEPLOY_DIR}/${INVENTORY}" ]]; then
    echo -e "${RED}❌ Inventory file not found: ${INVENTORY}${NC}"
    echo "   Edit the file and add your server IP."
    exit 1
fi

# Check playbook exists
if [[ ! -f "${DEPLOY_DIR}/${PLAYBOOK}" ]]; then
    echo -e "${RED}❌ Playbook not found: ${PLAYBOOK}${NC}"
    exit 1
fi

# ── Display Info ─────────────────────────────────────────────
echo -e "${YELLOW}📋 Configuration:${NC}"
echo -e "   Inventory: ${INVENTORY}"
echo -e "   Playbook:  ${PLAYBOOK}"
echo -e "   Extra:     ${EXTRA_VARS[*]:-none}"
echo ""

# ── Confirmation ─────────────────────────────────────────────
read -p "$(echo -e ${YELLOW}"▶ Proceed with deployment? (y/N): "${NC})" confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo -e "${RED}❌ Deployment cancelled.${NC}"
    exit 0
fi

# ── Run Ansible ──────────────────────────────────────────────
echo ""
echo -e "${GREEN}🚀 Starting deployment...${NC}"
echo ""

cd "${DEPLOY_DIR}"

ansible_cmd=(ansible-playbook -i "${INVENTORY}" "${PLAYBOOK}")
if [ ${#TAGS[@]} -gt 0 ]; then
    ansible_cmd+=("${TAGS[@]}")
fi
if [ ${#EXTRA_VARS[@]} -gt 0 ]; then
    ansible_cmd+=("${EXTRA_VARS[@]}")
fi

"${ansible_cmd[@]}"

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ✅ Deployment finished!                     ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
