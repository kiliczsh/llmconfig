#!/usr/bin/env bash
set -euo pipefail

# --- config ---
REPO="https://github.com/kiliczsh/llamaconfig.git"
BINARY_NAME="llamaconfig"
DEFAULT_PREFIX="/usr/local/bin"
MIN_DISK_MB=600

# --- flags ---
PREFIX="$DEFAULT_PREFIX"
NO_LLAMA=false
UPDATE=false

for arg in "$@"; do
  case "$arg" in
    --prefix=*) PREFIX="${arg#*=}" ;;
    --no-llama) NO_LLAMA=true ;;
    --update)   UPDATE=true ;;
    --help)
      echo "Usage: install.sh [--prefix=PATH] [--no-llama] [--update]"
      exit 0
      ;;
  esac
done

# --- colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# --- spinner ---
SPINNER_PID=""
spinner_chars='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'

spinner_start() {
  local msg="$1"
  (
    local i=0
    while true; do
      local c="${spinner_chars:$((i % ${#spinner_chars})):1}"
      printf "\r  ${CYAN}%s${RESET} %s " "$c" "$msg"
      sleep 0.08
      ((i++))
    done
  ) &
  SPINNER_PID=$!
}

spinner_stop() {
  if [[ -n "$SPINNER_PID" ]]; then
    kill "$SPINNER_PID" 2>/dev/null || true
    wait "$SPINNER_PID" 2>/dev/null || true
    SPINNER_PID=""
    printf "\r\033[K"
  fi
}

step_ok()   { printf "  ${GREEN}✓${RESET} %s\n" "$1"; }
step_fail() { printf "  ${RED}✗${RESET} %s\n" "$1"; }
step_info() { printf "  ${CYAN}→${RESET} %s\n" "$1"; }
step_warn() { printf "  ${YELLOW}!${RESET} %s\n" "$1"; }

die() {
  spinner_stop
  step_fail "$1"
  exit 1
}

trap 'spinner_stop' EXIT

# --- banner ---
banner() {
  echo ""
  printf "${CYAN}${BOLD}"
  cat <<'EOF'
  ██╗      ██╗      █████╗ ███╗   ███╗ █████╗  ██████╗ ██████╗ ███╗  ██╗███████╗██╗ ██████╗
  ██║      ██║     ██╔══██╗████╗ ████║██╔══██╗██╔════╝██╔═══██╗████╗ ██║██╔════╝██║██╔════╝
  ██║      ██║     ███████║██╔████╔██║███████║██║     ██║   ██║██╔██╗██║█████╗  ██║██║  ███╗
  ██║      ██║     ██╔══██║██║╚██╔╝██║██╔══██║██║     ██║   ██║██║╚████║██╔══╝  ██║██║   ██║
  ███████╗ ███████╗██║  ██║██║ ╚═╝ ██║██║  ██║╚██████╗╚██████╔╝██║ ╚███║██║     ██║╚██████╔╝
  ╚══════╝ ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚═╝  ╚══╝╚═╝     ╚═╝ ╚═════╝
EOF
  printf "${RESET}"
  printf "  %s\n\n" "Manage local LLM inference with llama.cpp"
}

# --- step counter ---
STEP=0
TOTAL_STEPS=5
[[ "$NO_LLAMA" == false ]] && TOTAL_STEPS=6

step() {
  ((STEP++))
  printf "\n${BOLD}[%d/%d] %s${RESET}\n" "$STEP" "$TOTAL_STEPS" "$1"
}

# --- pre-flight animation ---
preflight_animate() {
  local checks=("OS" "Disk" "Internet" "Git" "Go")
  printf "\n  ${BOLD}Pre-flight checks${RESET}\n"
  for c in "${checks[@]}"; do
    printf "  ${CYAN}...${RESET} %-10s" "$c"
    sleep 0.12
    printf "\r  ${GREEN}✓${RESET}   %-10s\n" "$c"
  done
}

# ============================================================
# MAIN
# ============================================================
banner

# --- [1] Pre-flight ---
step "Pre-flight checks"

# OS
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS" in
  Darwin|Linux) step_ok "OS: $OS/$ARCH" ;;
  *) die "Unsupported OS: $OS" ;;
esac

# Disk
available_mb=$(df -m "$HOME" | awk 'NR==2{print $4}')
if (( available_mb < MIN_DISK_MB )); then
  die "Not enough disk space (need ${MIN_DISK_MB}MB, have ${available_mb}MB)"
fi
step_ok "Disk: ${available_mb}MB available"

# Internet
if ! curl -sf --max-time 5 https://github.com > /dev/null 2>&1; then
  die "No internet connection"
fi
step_ok "Internet: reachable"

# Git
if ! command -v git &>/dev/null; then
  die "git not found — install git first"
fi
step_ok "Git: $(git --version | awk '{print $3}')"

# Go
if ! command -v go &>/dev/null; then
  # try common brew paths
  for p in /opt/homebrew/bin/go /usr/local/go/bin/go; do
    [[ -x "$p" ]] && export PATH="$(dirname $p):$PATH" && break
  done
fi

if ! command -v go &>/dev/null; then
  step_warn "Go not found — installing via brew..."
  if ! command -v brew &>/dev/null; then
    die "Homebrew not found. Install Go manually: https://go.dev/doc/install"
  fi
  brew install go
fi
step_ok "Go: $(go version | awk '{print $3}')"

# --- [2] Clone / Update ---
step "Repository"

INSTALL_DIR="$HOME/.llamaconfig/src"
mkdir -p "$(dirname "$INSTALL_DIR")"

if [[ -d "$INSTALL_DIR/.git" ]]; then
  spinner_start "Updating repository..."
  git -C "$INSTALL_DIR" pull --ff-only > /dev/null 2>&1
  spinner_stop
  step_ok "Repository updated"
else
  spinner_start "Cloning repository..."
  git clone --depth=1 "$REPO" "$INSTALL_DIR" > /dev/null 2>&1
  spinner_stop
  step_ok "Repository cloned to $INSTALL_DIR"
fi

# --- [3] Build ---
step "Building llamaconfig"

spinner_start "Compiling..."
(cd "$INSTALL_DIR" && go build -o llamaconfig . > /dev/null 2>&1)
spinner_stop
step_ok "Build successful"

# --- [4] Install binary ---
step "Installing binary"

mkdir -p "$PREFIX" 2>/dev/null || true

if ! cp "$INSTALL_DIR/llamaconfig" "$PREFIX/$BINARY_NAME" 2>/dev/null; then
  # fallback: try sudo, then ~/.local/bin
  if command -v sudo &>/dev/null && sudo cp "$INSTALL_DIR/llamaconfig" "$PREFIX/$BINARY_NAME" 2>/dev/null; then
    sudo chmod +x "$PREFIX/$BINARY_NAME"
    step_ok "Installed to $PREFIX/$BINARY_NAME (via sudo)"
  else
    PREFIX="$HOME/.local/bin"
    mkdir -p "$PREFIX"
    cp "$INSTALL_DIR/llamaconfig" "$PREFIX/$BINARY_NAME"
    chmod +x "$PREFIX/$BINARY_NAME"
    step_warn "No permission for /usr/local/bin — installed to $PREFIX/$BINARY_NAME"
  fi
else
  chmod +x "$PREFIX/$BINARY_NAME"
  step_ok "Installed to $PREFIX/$BINARY_NAME"
fi

# lc alias
if ln -sf "$PREFIX/$BINARY_NAME" "$PREFIX/lc" 2>/dev/null || sudo ln -sf "$PREFIX/$BINARY_NAME" "$PREFIX/lc" 2>/dev/null; then
  step_ok "Alias: lc → llamaconfig"
else
  step_warn "Could not create lc alias in $PREFIX"
fi

# PATH check
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$PREFIX"; then
  step_warn "$PREFIX is not in your PATH"
  SHELL_RC=""
  case "$SHELL" in
    */zsh)  SHELL_RC="$HOME/.zshrc" ;;
    */bash) SHELL_RC="$HOME/.bashrc" ;;
  esac
  if [[ -n "$SHELL_RC" ]]; then
    echo "" >> "$SHELL_RC"
    echo "export PATH=\"$PREFIX:\$PATH\"" >> "$SHELL_RC"
    step_ok "Added $PREFIX to PATH in $SHELL_RC (restart shell to apply)"
  else
    step_warn "Add $PREFIX to your PATH manually"
  fi
fi

# --- [5] Smoke test ---
step "Smoke test"

VERSION="$("$PREFIX/$BINARY_NAME" version 2>/dev/null || echo 'unknown')"
step_ok "llamaconfig $VERSION"

HW="$("$PREFIX/$BINARY_NAME" hardware 2>/dev/null | grep 'Selected profile' | awk -F': ' '{print $2}' | xargs)"
step_ok "Hardware profile: ${HW:-detected}"

# --- [6] Install llama.cpp ---
if [[ "$NO_LLAMA" == false ]]; then
  step "Installing llama.cpp"
  spinner_start "Downloading llama.cpp binary..."
  "$PREFIX/$BINARY_NAME" llama --install < /dev/null 2>&1 | grep -E '^(->|./)' || true
  spinner_stop
  LLAMA_VERSION="$("$PREFIX/$BINARY_NAME" llama --version 2>/dev/null | grep 'version:' | head -1 || echo 'installed')"
  step_ok "llama.cpp: $LLAMA_VERSION"
fi

# --- done ---
echo ""
printf "${GREEN}${BOLD}  Installation complete!${RESET}\n\n"
printf "  Run: ${CYAN}lc init --template gemma${RESET}\n"
printf "       ${CYAN}lc up <model-name>${RESET}\n"
echo ""
