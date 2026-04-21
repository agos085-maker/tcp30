#!/bin/bash
# ─────────────────────────────────────────────────────────────
#  Kost App - Startup Script
# ─────────────────────────────────────────────────────────────

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

echo ""
echo -e "${CYAN}${BOLD}  🏠  KOST APP - Manajemen Kost${NC}"
echo -e "  ─────────────────────────────────"
echo ""

# Check dependencies
check_dep() {
  if ! command -v "$1" &> /dev/null; then
    echo -e "${RED}✗ '$1' tidak ditemukan.${NC}"
    echo -e "  Install: $2"
    exit 1
  fi
  echo -e "${GREEN}✓${NC} $1 tersedia"
}

echo -e "${BOLD}Memeriksa dependencies...${NC}"
check_dep "sqlite3"  "sudo apt install sqlite3  /  brew install sqlite3"
check_dep "python3"  "https://python.org"

# Check fpdf2
if ! python3 -c "import fpdf" 2>/dev/null; then
  echo -e "${YELLOW}⚠ fpdf2 belum terinstall. Menginstall...${NC}"
  pip install fpdf2 --break-system-packages 2>/dev/null || pip install fpdf2
fi
echo -e "${GREEN}✓${NC} fpdf2 tersedia"

# Build if binary not present
if [ ! -f "./kost-app" ]; then
  echo ""
  echo -e "${BOLD}Building kost-app...${NC}"
  if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ Go tidak ditemukan. Install dari https://go.dev/dl/${NC}"
    exit 1
  fi
  go build -o kost-app . && echo -e "${GREEN}✓${NC} Build berhasil"
fi

echo ""
echo -e "${GREEN}${BOLD}✅ Semua siap!${NC}"
echo -e "${CYAN}   Buka browser: http://localhost:8080${NC}"
echo -e "   Tekan Ctrl+C untuk berhenti"
echo ""

# Run
./kost-app
