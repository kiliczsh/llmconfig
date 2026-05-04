#!/usr/bin/env bash
set -euo pipefail

# --- config ---
REPO_SLUG="kiliczsh/llmconfig"
BINARY_NAME="llmconfig"
DEFAULT_PREFIX="/usr/local/bin"
MIN_DISK_MB=200

# --- flags ---
PREFIX="$DEFAULT_PREFIX"
NO_BACKENDS=false
UPDATE=false
VERSION=""

for arg in "$@"; do
  case "$arg" in
    --prefix=*)    PREFIX="${arg#*=}" ;;
    --version=*)   VERSION="${arg#*=}" ;;
    --no-llama)    NO_BACKENDS=true ;;
    --no-backends) NO_BACKENDS=true ;;
    --update)      UPDATE=true ;;
    --help)
      cat <<USAGE
Usage: install.sh [--prefix=PATH] [--version=vX.Y.Z] [--no-backends] [--update]

  --prefix=PATH    Install directory (default: /usr/local/bin)
  --version=TAG    Install a specific release tag (default: latest)
  --no-backends    Skip backend downloads (llama.cpp, sd.cpp, whisper.cpp)
  --update         Force reinstall even if already present

The script downloads a prebuilt binary from GitHub Releases; no Go or
git toolchain is required on the target machine.

When run from inside an extracted release archive (i.e. the llmconfig
binary sits next to this script), the download step is skipped and the
adjacent binary is installed directly — useful for offline installs.
USAGE
      exit 0
      ;;
  esac
done

# Detect whether we're being run from inside an already-extracted release
# archive. When piped via `curl ... | bash`, BASH_SOURCE[0] is empty or
# non-resolvable, so this falls through to the download path.
SCRIPT_DIR=""
if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd || echo "")"
fi
LOCAL_MODE=false
if [[ -n "$SCRIPT_DIR" && -f "$SCRIPT_DIR/$BINARY_NAME" && -z "$VERSION" ]]; then
  LOCAL_MODE=true
fi

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
  ██╗     ██╗     ███╗   ███╗ ██████╗ ██████╗ ███╗   ██╗███████╗██╗ ██████╗
  ██║     ██║     ████╗ ████║██╔════╝██╔═══██╗████╗  ██║██╔════╝██║██╔════╝
  ██║     ██║     ██╔████╔██║██║     ██║   ██║██╔██╗ ██║█████╗  ██║██║  ███╗
  ██║     ██║     ██║╚██╔╝██║██║     ██║   ██║██║╚██╗██║██╔══╝  ██║██║   ██║
  ███████╗███████╗██║ ╚═╝ ██║╚██████╗╚██████╔╝██║ ╚████║██║     ██║╚██████╔╝
  ╚══════╝╚══════╝╚═╝     ╚═╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝     ╚═╝ ╚═════╝
EOF
  printf "${RESET}"
  printf "  %s\n\n" "Local Large Model Config — llama.cpp · sd.cpp · whisper.cpp"
}

STEP=0
# Remote mode: pre-flight, resolve, download, install, (llama)
# Local mode:  pre-flight, use-bundled, install, (llama)
if [[ "$LOCAL_MODE" == true ]]; then
  TOTAL_STEPS=3
else
  TOTAL_STEPS=4
fi
if [[ "$NO_BACKENDS" == false ]]; then
  TOTAL_STEPS=$(( TOTAL_STEPS + 1 ))
fi

step() {
  STEP=$(( STEP + 1 ))
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

if ! command -v curl &>/dev/null; then
  die "curl not found — install curl first"
fi
if ! command -v tar &>/dev/null; then
  die "tar not found — install tar first"
fi

# In local mode we still want the network for llama.cpp install later,
# but don't fail here if offline — `--no-llama` is a valid path.
if [[ "$LOCAL_MODE" == false || "$NO_BACKENDS" == false ]]; then
  if ! curl -sf --max-time 5 https://github.com > /dev/null 2>&1; then
    if [[ "$LOCAL_MODE" == true ]]; then
      step_warn "No internet — --no-backends will be forced after install"
      NO_BACKENDS=true
      TOTAL_STEPS=$(( TOTAL_STEPS - 1 ))
    else
      die "No internet connection"
    fi
  else
    step_ok "Internet: reachable"
  fi
fi

# SRC_DIR is where install step reads binaries from: either a fresh
# tmpdir (remote mode, filled by the download+extract step) or the
# script's own directory (local mode, the extracted release archive).
SRC_DIR=""

if [[ "$LOCAL_MODE" == true ]]; then
  # --- [2] Use bundled binary ---
  step "Using bundled binary"
  SRC_DIR="$SCRIPT_DIR"
  VERSION_NO_V=$("$SRC_DIR/$BINARY_NAME" version 2>/dev/null | awk '{print $2}' || echo "bundled")
  step_ok "Version: $VERSION_NO_V (from $SRC_DIR)"

  if [[ "$UPDATE" == false ]] && command -v "$BINARY_NAME" &>/dev/null; then
    INSTALLED=$("$BINARY_NAME" version 2>/dev/null | awk '{print $2}' || true)
    if [[ -n "$INSTALLED" && "$INSTALLED" == "$VERSION_NO_V" ]]; then
      step_ok "$BINARY_NAME $INSTALLED is already installed (use --update to reinstall)"
      [[ "$NO_BACKENDS" == true ]] && exit 0
    fi
  fi
else
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

  if [[ "$UPDATE" == false ]] && command -v "$BINARY_NAME" &>/dev/null; then
    INSTALLED=$("$BINARY_NAME" version 2>/dev/null | awk '{print $2}' || true)
    if [[ -n "$INSTALLED" && "$INSTALLED" == "$VERSION_NO_V" ]]; then
      step_ok "$BINARY_NAME $INSTALLED is already installed (use --update to reinstall)"
      [[ "$NO_BACKENDS" == true ]] && exit 0
    fi
  fi

  ARCHIVE="llmconfig-${VERSION_NO_V}-${OS}-${ARCH}.tar.gz"
  ARCHIVE_URL="https://github.com/${REPO_SLUG}/releases/download/${VERSION}/${ARCHIVE}"
  CHECKSUM_URL="https://github.com/${REPO_SLUG}/releases/download/${VERSION}/checksums.txt"

  # --- [3] Download & verify ---
  step "Downloading binary"

  SRC_DIR="$(mktemp -d)"
  trap 'spinner_stop; rm -rf "$SRC_DIR"' EXIT

  spinner_start "Downloading $ARCHIVE..."
  if ! curl -sSLf "$ARCHIVE_URL" -o "$SRC_DIR/$ARCHIVE"; then
    spinner_stop
    die "Download failed: $ARCHIVE_URL"
  fi
  spinner_stop
  step_ok "Downloaded $(du -h "$SRC_DIR/$ARCHIVE" | cut -f1)"

  if curl -sSLf "$CHECKSUM_URL" -o "$SRC_DIR/checksums.txt" 2>/dev/null; then
    EXPECTED=$(grep "  $ARCHIVE\$" "$SRC_DIR/checksums.txt" | awk '{print $1}')
    if [[ -n "$EXPECTED" ]]; then
      if command -v sha256sum &>/dev/null; then
        ACTUAL=$(sha256sum "$SRC_DIR/$ARCHIVE" | awk '{print $1}')
      elif command -v shasum &>/dev/null; then
        ACTUAL=$(shasum -a 256 "$SRC_DIR/$ARCHIVE" | awk '{print $1}')
      fi
      if [[ -n "${ACTUAL:-}" && "$ACTUAL" != "$EXPECTED" ]]; then
        die "Checksum mismatch (expected $EXPECTED, got $ACTUAL)"
      fi
      step_ok "Checksum verified"
    fi
  else
    step_warn "No checksum file — skipping verification"
  fi

  tar -xzf "$SRC_DIR/$ARCHIVE" -C "$SRC_DIR"
  if [[ ! -f "$SRC_DIR/$BINARY_NAME" ]]; then
    die "Archive did not contain $BINARY_NAME"
  fi
fi

# --- Install ---
step "Installing binary"

mkdir -p "$PREFIX" 2>/dev/null || true
DEST="$PREFIX/$BINARY_NAME"

install_binary() {
  local use_sudo="$1"
  local cp_cmd="cp"
  local ln_cmd="ln"
  local chmod_cmd="chmod"
  if [[ "$use_sudo" == "sudo" ]]; then
    cp_cmd="sudo cp"
    ln_cmd="sudo ln"
    chmod_cmd="sudo chmod"
  fi
  $cp_cmd "$SRC_DIR/$BINARY_NAME" "$DEST" && $chmod_cmd +x "$DEST" || return 1
  # Prefer the bundled llmc binary from the archive; fall back to a symlink
  # for older releases that don't include it.
  if [[ -f "$SRC_DIR/llmc" ]]; then
    $cp_cmd "$SRC_DIR/llmc" "$PREFIX/llmc" && $chmod_cmd +x "$PREFIX/llmc"
  else
    $ln_cmd -sf "$DEST" "$PREFIX/llmc" 2>/dev/null || true
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

# macOS Gatekeeper: clear quarantine flag so the binary runs without prompts
if [[ "$OS" == "darwin" ]]; then
  for _bin in "$DEST" "$PREFIX/llmc"; do
    if [[ -f "$_bin" ]]; then
      xattr -d com.apple.quarantine "$_bin" 2>/dev/null || \
        sudo xattr -d com.apple.quarantine "$_bin" 2>/dev/null || true
    fi
  done
fi

# Smoke test
INSTALLED_VERSION=$("$DEST" version 2>/dev/null || echo 'unknown')
step_ok "$INSTALLED_VERSION"

# Copy bundled templates to configs dir (skip existing)
TEMPLATES_DIR="$(dirname "$0")/templates"
if [[ -d "$TEMPLATES_DIR" ]]; then
    CONFIGS_DIR="$HOME/.llmconfig/configs"
    mkdir -p "$CONFIGS_DIR"
    for f in "$TEMPLATES_DIR"/*.llmc; do
        [[ -e "$f" ]] || continue
        dest="$CONFIGS_DIR/$(basename "$f")"
        # Skip if either extension is already present so existing
        # users keep their tweaks (the in-process migrator handles
        # the .yaml→.llmc rename for old installs).
        base="$(basename "${f%.llmc}")"
        if [[ ! -f "$dest" && ! -f "$CONFIGS_DIR/$base.yaml" ]]; then
            cp "$f" "$dest"
            step_ok "Template: $(basename "$f")"
        fi
    done
fi

# --- [5] Install backends (llama.cpp, stable-diffusion.cpp, whisper.cpp) ---
if [[ "$NO_BACKENDS" == false ]]; then
  step "Installing backends"

  # Resolve the binary to use for backend installs: prefer the installed $DEST,
  # fall back to the source binary in local mode (dev builds may not install cleanly).
  LC_BIN=""
  if [[ -x "$DEST" ]]; then
    LC_BIN="$DEST"
  elif [[ "$LOCAL_MODE" == true && -x "$SRC_DIR/$BINARY_NAME" ]]; then
    LC_BIN="$SRC_DIR/$BINARY_NAME"
  else
    step_warn "llmconfig binary not runnable at $DEST — skipping backend installs"
  fi

  if [[ -n "$LC_BIN" ]]; then

  # llama.cpp
  LLAMA_BIN="$("$LC_BIN" llama --path 2>/dev/null || true)"
  if [[ -n "$LLAMA_BIN" && -f "$LLAMA_BIN" && "$UPDATE" == false ]]; then
    LLAMA_VERSION="$("$LC_BIN" llama --version 2>/dev/null | grep 'version:' | head -1 || echo 'unknown')"
    step_ok "llama.cpp already installed: ${LLAMA_VERSION}"
  else
    spinner_start "Downloading llama.cpp..."
    "$LC_BIN" install llama < /dev/null 2>&1 || true
    spinner_stop
    LLAMA_VERSION="$("$LC_BIN" llama --version 2>/dev/null | grep 'version:' | head -1 || echo 'installed')"
    step_ok "llama.cpp: $LLAMA_VERSION"
  fi

  # stable-diffusion.cpp
  SD_BIN="$("$LC_BIN" sd --path 2>/dev/null || true)"
  if [[ -n "$SD_BIN" && -f "$SD_BIN" && "$UPDATE" == false ]]; then
    SD_VERSION="$("$LC_BIN" sd --version 2>/dev/null | grep 'version:' | head -1 || echo 'unknown')"
    step_ok "stable-diffusion.cpp already installed: ${SD_VERSION}"
  else
    spinner_start "Downloading stable-diffusion.cpp..."
    "$LC_BIN" install sd < /dev/null 2>&1 || true
    spinner_stop
    SD_VERSION="$("$LC_BIN" sd --version 2>/dev/null | grep 'version:' | head -1 || echo 'installed')"
    step_ok "stable-diffusion.cpp: $SD_VERSION"
  fi

  # whisper.cpp (pre-built binaries only available on Windows)
  if [[ "$OS" != "linux" && "$OS" != "darwin" ]]; then
    WHISPER_BIN="$("$LC_BIN" whisper --path 2>/dev/null || true)"
    if [[ -n "$WHISPER_BIN" && -f "$WHISPER_BIN" && "$UPDATE" == false ]]; then
      WHISPER_VERSION="$("$LC_BIN" whisper --version 2>/dev/null | grep 'version:' | head -1 || echo 'unknown')"
      step_ok "whisper.cpp already installed: ${WHISPER_VERSION}"
    else
      spinner_start "Downloading whisper.cpp..."
      "$LC_BIN" install whisper < /dev/null 2>&1 || true
      spinner_stop
      WHISPER_VERSION="$("$LC_BIN" whisper --version 2>/dev/null | grep 'version:' | head -1 || echo 'installed')"
      step_ok "whisper.cpp: $WHISPER_VERSION"
    fi
  else
    if [[ "$OS" == "darwin" ]]; then
      if command -v whisper-server &>/dev/null || command -v whisper-cli &>/dev/null; then
        step_ok "whisper.cpp already installed: $(command -v whisper-server whisper-cli 2>/dev/null | head -1)"
      elif command -v brew &>/dev/null; then
        step_warn "whisper.cpp: no pre-built binaries for macOS — installing via Homebrew..."
        if brew install whisper-cpp &>/dev/null; then
          step_ok "whisper.cpp installed via Homebrew"
        else
          step_warn "whisper.cpp: brew install failed — try manually: brew install whisper-cpp"
        fi
      else
        step_warn "whisper.cpp: no pre-built binaries for macOS — install via: brew install whisper-cpp"
      fi
    else
      step_warn "whisper.cpp: no pre-built binaries for $OS — build from source: https://github.com/ggml-org/whisper.cpp"
    fi
  fi

  fi # end LC_BIN check
fi

echo ""
printf "${GREEN}${BOLD}  Installation complete!${RESET}\n\n"
printf "  Run: ${CYAN}llmc init --template gemma${RESET}\n"
printf "       ${CYAN}llmc up <model-name>${RESET}\n"
echo ""
