#!/usr/bin/env bash
set -euo pipefail

# --- config ---
REPO_SLUG="kiliczsh/llamaconfig"
BINARY_NAME="llamaconfig"
DEFAULT_PREFIX="/usr/local/bin"
MIN_DISK_MB=200

# --- flags ---
PREFIX="$DEFAULT_PREFIX"
NO_LLAMA=false
UPDATE=false
VERSION=""

for arg in "$@"; do
  case "$arg" in
    --prefix=*)  PREFIX="${arg#*=}" ;;
    --version=*) VERSION="${arg#*=}" ;;
    --no-llama)  NO_LLAMA=true ;;
    --update)    UPDATE=true ;;
    --help)
      cat <<USAGE
Usage: install.sh [--prefix=PATH] [--version=vX.Y.Z] [--no-llama] [--update]

  --prefix=PATH    Install directory (default: /usr/local/bin)
  --version=TAG    Install a specific release tag (default: latest)
  --no-llama       Skip llama.cpp download
  --update         Force reinstall even if already present

The script downloads a prebuilt binary from GitHub Releases; no Go or
git toolchain is required on the target machine.
USAGE
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

STEP=0
TOTAL_STEPS=4
[[ "$NO_LLAMA" == false ]] && TOTAL_STEPS=5

step() {
  ((STEP++))
  printf "\n${BOLD}[%d/%d] %s${RESET}\n" "$STEP" "$TOTAL_STEPS" "$1"
}

# ============================================================
# MAIN
# ============================================================
banner

# --- [1] Pre-flight ---
step "Pre-flight checks"

OS_RAW="$(uname -s)"
ARCH_RAW="$(uname -m)"
case "$OS_RAW" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux"  ;;
  *)      die "Unsupported OS: $OS_RAW" ;;
esac
case "$ARCH_RAW" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             die "Unsupported arch: $ARCH_RAW" ;;
esac
step_ok "OS: $OS/$ARCH"

available_mb=$(df -m "$HOME" | awk 'NR==2{print $4}')
if (( available_mb < MIN_DISK_MB )); then
  die "Not enough disk space (need ${MIN_DISK_MB}MB, have ${available_mb}MB)"
fi
step_ok "Disk: ${available_mb}MB available"

if ! curl -sf --max-time 5 https://github.com > /dev/null 2>&1; then
  die "No internet connection"
fi
step_ok "Internet: reachable"

if ! command -v curl &>/dev/null; then
  die "curl not found — install curl first"
fi
if ! command -v tar &>/dev/null; then
  die "tar not found — install tar first"
fi

# --- [2] Resolve release ---
step "Resolving release"

if [[ -z "$VERSION" ]]; then
  spinner_start "Fetching latest release tag..."
  VERSION=$(curl -sSL "https://api.github.com/repos/${REPO_SLUG}/releases/latest" \
    | sed -n 's/.*"tag_name": *"\(v[^"]*\)".*/\1/p' | head -1)
  spinner_stop
  [[ -z "$VERSION" ]] && die "Could not determine latest release (API rate-limited? Try --version=vX.Y.Z)"
fi
VERSION_NO_V="${VERSION#v}"
step_ok "Version: $VERSION"

# Skip reinstall if same version already present
if [[ "$UPDATE" == false ]] && command -v "$BINARY_NAME" &>/dev/null; then
  INSTALLED=$("$BINARY_NAME" version 2>/dev/null | awk '{print $2}' || true)
  if [[ -n "$INSTALLED" && "$INSTALLED" == "$VERSION_NO_V" ]]; then
    step_ok "$BINARY_NAME $INSTALLED is already installed (use --update to reinstall)"
    [[ "$NO_LLAMA" == true ]] && exit 0
  fi
fi

ARCHIVE="llamaconfig-${VERSION_NO_V}-${OS}-${ARCH}.tar.gz"
ARCHIVE_URL="https://github.com/${REPO_SLUG}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${REPO_SLUG}/releases/download/${VERSION}/checksums.txt"

# --- [3] Download & verify ---
step "Downloading binary"

TMP="$(mktemp -d)"
trap 'spinner_stop; rm -rf "$TMP"' EXIT

spinner_start "Downloading $ARCHIVE..."
if ! curl -sSLf "$ARCHIVE_URL" -o "$TMP/$ARCHIVE"; then
  spinner_stop
  die "Download failed: $ARCHIVE_URL"
fi
spinner_stop
step_ok "Downloaded $(du -h "$TMP/$ARCHIVE" | cut -f1)"

# Verify checksum — non-fatal if checksums.txt missing (older releases)
if curl -sSLf "$CHECKSUM_URL" -o "$TMP/checksums.txt" 2>/dev/null; then
  EXPECTED=$(grep "  $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')
  if [[ -n "$EXPECTED" ]]; then
    if command -v sha256sum &>/dev/null; then
      ACTUAL=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
      ACTUAL=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
    fi
    if [[ -n "${ACTUAL:-}" && "$ACTUAL" != "$EXPECTED" ]]; then
      die "Checksum mismatch (expected $EXPECTED, got $ACTUAL)"
    fi
    step_ok "Checksum verified"
  fi
else
  step_warn "No checksum file — skipping verification"
fi

# --- [4] Install ---
step "Installing binary"

tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
if [[ ! -f "$TMP/$BINARY_NAME" ]]; then
  die "Archive did not contain $BINARY_NAME"
fi

mkdir -p "$PREFIX" 2>/dev/null || true
DEST="$PREFIX/$BINARY_NAME"

install_binary() {
  local use_sudo="$1"
  if [[ "$use_sudo" == "sudo" ]]; then
    sudo cp "$TMP/$BINARY_NAME" "$DEST" && sudo chmod +x "$DEST"
    sudo ln -sf "$DEST" "$PREFIX/lc" 2>/dev/null || true
  else
    cp "$TMP/$BINARY_NAME" "$DEST" && chmod +x "$DEST"
    ln -sf "$DEST" "$PREFIX/lc" 2>/dev/null || true
  fi
}

if install_binary "" 2>/dev/null; then
  step_ok "Installed to $DEST"
elif command -v sudo &>/dev/null && install_binary "sudo" 2>/dev/null; then
  step_ok "Installed to $DEST (via sudo)"
else
  PREFIX="$HOME/.local/bin"
  DEST="$PREFIX/$BINARY_NAME"
  mkdir -p "$PREFIX"
  install_binary "" || die "Failed to install to $PREFIX"
  step_warn "No permission for $DEFAULT_PREFIX — installed to $DEST"
fi

# PATH check
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$PREFIX"; then
  step_warn "$PREFIX is not in your PATH"
  SHELL_RC=""
  case "${SHELL:-}" in
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

# Smoke test
INSTALLED_VERSION=$("$DEST" version 2>/dev/null || echo 'unknown')
step_ok "$INSTALLED_VERSION"

# --- [5] Install llama.cpp ---
if [[ "$NO_LLAMA" == false ]]; then
  step "Installing llama.cpp"
  LLAMA_BIN="$("$DEST" llama --path 2>/dev/null || true)"
  if [[ -n "$LLAMA_BIN" && -f "$LLAMA_BIN" && "$UPDATE" == false ]]; then
    LLAMA_VERSION="$("$DEST" llama --version 2>/dev/null | grep 'version:' | head -1 || echo 'unknown')"
    step_ok "llama.cpp already installed: ${LLAMA_VERSION} (use --update to reinstall)"
  else
    spinner_start "Downloading llama.cpp binary..."
    "$DEST" llama --install < /dev/null 2>&1 | grep -E '^(->|./)' || true
    spinner_stop
    LLAMA_VERSION="$("$DEST" llama --version 2>/dev/null | grep 'version:' | head -1 || echo 'installed')"
    step_ok "llama.cpp: $LLAMA_VERSION"
  fi
fi

echo ""
printf "${GREEN}${BOLD}  Installation complete!${RESET}\n\n"
printf "  Run: ${CYAN}lc init --template gemma${RESET}\n"
printf "       ${CYAN}lc up <model-name>${RESET}\n"
echo ""
