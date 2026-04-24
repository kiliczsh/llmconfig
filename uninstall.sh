#!/usr/bin/env bash
set -euo pipefail

# --- config ---
BINARY_NAME="llmconfig"
DEFAULT_PREFIX="/usr/local/bin"
LOCAL_PREFIX="$HOME/.local/bin"
SRC_DIR="$HOME/.llmconfig/src"
CONFIG_DIR="$HOME/.llmconfig/configs"
BIN_DIR="$HOME/.llmconfig/bin"
LOGS_DIR="$HOME/.llmconfig/logs"
MODELS_DIR="$HOME/.llmconfig/models"
LLMCONFIG_HOME="$HOME/.llmconfig"

# --- colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

step_ok()   { printf "  ${GREEN}✓${RESET} %s\n" "$1"; }
step_skip() { printf "  ${DIM}-${RESET} %s\n" "$1"; }

# --- find installed binary prefix ---
PREFIX="$DEFAULT_PREFIX"
if [[ ! -f "$PREFIX/$BINARY_NAME" ]] && [[ -f "$LOCAL_PREFIX/$BINARY_NAME" ]]; then
  PREFIX="$LOCAL_PREFIX"
fi

# ------------------------------------------------------------------ #
# Interactive checkbox selector
# Usage: checkbox_select "Title" item1 item2 ...
# Sets global SELECTED array with indices of checked items (1-based)
# ------------------------------------------------------------------ #
declare -a SELECTED=()

checkbox_select() {
  local title="$1"
  shift
  local items=("$@")
  local count=${#items[@]}
  local cursor=0
  local -a checked=()

  # default: all checked
  for (( i=0; i<count; i++ )); do checked[$i]=1; done

  # hide cursor
  tput civis 2>/dev/null || true
  trap 'tput cnorm 2>/dev/null || true' EXIT

  draw() {
    local mode="${1:-}"
    # move up 'count+2' lines to redraw
    if [[ "$mode" != "first" ]]; then
      tput cuu $(( count + 2 )) 2>/dev/null || true
    fi
    printf "  ${BOLD}%s${RESET}\n" "$title"
    printf "  ${DIM}Space: toggle  Enter: confirm  a: all  n: none${RESET}\n"
    for (( i=0; i<count; i++ )); do
      local box="${DIM}[ ]${RESET}"
      [[ "${checked[$i]}" -eq 1 ]] && box="${GREEN}[x]${RESET}" || true
      local label="${items[$i]}"
      if [[ "$i" -eq "$cursor" ]]; then
        printf "  ${CYAN}>${RESET} %b  ${BOLD}%s${RESET}\n" "$box" "$label"
      else
        printf "    %b  %s\n" "$box" "$label"
      fi
    done
  }

  draw first
  while true; do
    # read a single keypress (incl. escape sequences)
    IFS= read -rsn1 key
    if [[ "$key" == $'\x1b' ]]; then
      local key2=""
      read -rsn2 -t 1 key2 || true
      key+="$key2"
    fi

    case "$key" in
      $'\x1b[A'|k)  [[ "$cursor" -gt 0 ]] && cursor=$(( cursor - 1 )) || true ;;
      $'\x1b[B'|j)  [[ "$cursor" -lt $(( count - 1 )) ]] && cursor=$(( cursor + 1 )) || true ;;
      ' ')          checked[$cursor]=$(( 1 - checked[$cursor] )) || true ;;
      a|A)          for (( i=0; i<count; i++ )); do checked[$i]=1; done ;;
      n|N)          for (( i=0; i<count; i++ )); do checked[$i]=0; done ;;
      ''|$'\n')     break ;;
      q|Q)          tput cnorm 2>/dev/null || true; echo ""; printf "  Aborted.\n"; exit 0 ;;
    esac
    draw
  done

  tput cnorm 2>/dev/null || true
  SELECTED=()
  for (( i=0; i<count; i++ )); do
    if [[ "${checked[$i]}" -eq 1 ]]; then SELECTED+=("$i"); fi
  done
}

has_selected() {
  local idx="$1"
  for s in "${SELECTED[@]:-}"; do [[ "$s" -eq "$idx" ]] && return 0; done
  return 1
}

# ------------------------------------------------------------------ #
# Banner
# ------------------------------------------------------------------ #
echo ""
printf "${RED}${BOLD}"
cat <<'EOF'
  ██╗   ██╗███╗  ██╗██╗███╗  ██╗███████╗████████╗ █████╗ ██╗     ██╗
  ██║   ██║████╗ ██║██║████╗ ██║██╔════╝╚══██╔══╝██╔══██╗██║     ██║
  ██║   ██║██╔██╗██║██║██╔██╗██║███████╗   ██║   ███████║██║     ██║
  ██║   ██║██║╚████║██║██║╚████║╚════██║   ██║   ██╔══██║██║     ██║
  ╚██████╔╝██║ ╚███║██║██║ ╚███║███████║   ██║   ██║  ██║███████╗███████╗
   ╚═════╝ ╚═╝  ╚══╝╚═╝╚═╝  ╚══╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝
EOF
printf "${RESET}\n"

# ------------------------------------------------------------------ #
# Build item labels with sizes
# ------------------------------------------------------------------ #
_sz() { [[ -e "$1" ]] && du -sh "$1" 2>/dev/null | awk '{print $1}' || echo "-"; }

label_binary="llmconfig binary       ($PREFIX/$BINARY_NAME)"
label_llama="llama.cpp binaries       ($(_sz "$BIN_DIR"))"
label_src="Source code              ($(_sz "$SRC_DIR"))"
label_configs="Model configs            ($(_sz "$CONFIG_DIR"))"
label_logs="Logs                     ($(_sz "$LOGS_DIR"))"
label_models="Downloaded models (GGUF files) ($(_sz "$MODELS_DIR"))"

ITEMS=(
  "$label_binary"
  "$label_llama"
  "$label_src"
  "$label_configs"
  "$label_logs"
  "$label_models"
)

# cache unchecked by default - reorder so cache is last and uncheck it
checkbox_select "Select what to remove:" "${ITEMS[@]}"

# uncheck cache by default: if user didn't explicitly toggle it stays off
# (we start all checked so user must uncheck cache manually - that's the UX)

echo ""

if [[ ${#SELECTED[@]} -eq 0 ]]; then
  printf "  Nothing selected. Aborted.\n\n"
  exit 0
fi

# ------------------------------------------------------------------ #
# Confirm
# ------------------------------------------------------------------ #
printf "  ${BOLD}Remove ${#SELECTED[@]} item(s)?${RESET} [y/N] "
read -r confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
  printf "  Aborted.\n\n"
  exit 0
fi
echo ""

# ------------------------------------------------------------------ #
# Execute removals
# ------------------------------------------------------------------ #

# 0: binary
if has_selected 0; then
  # stop running models first
  if command -v "$PREFIX/$BINARY_NAME" &>/dev/null; then
    "$PREFIX/$BINARY_NAME" down --all 2>/dev/null || true
  fi
  if [[ -f "$PREFIX/$BINARY_NAME" ]]; then
    rm -f "$PREFIX/$BINARY_NAME" 2>/dev/null || sudo rm -f "$PREFIX/$BINARY_NAME"
    step_ok "Removed binary"
  else
    step_skip "Binary not found"
  fi
  if [[ -L "$PREFIX/llmc" || -f "$PREFIX/llmc" ]]; then
    rm -f "$PREFIX/llmc" 2>/dev/null || sudo rm -f "$PREFIX/llmc"
    step_ok "Removed llmc"
  fi
fi

# 1: llama.cpp
if has_selected 1; then
  if [[ -d "$BIN_DIR" ]]; then
    rm -rf "$BIN_DIR"
    step_ok "Removed llama.cpp binaries"
  else
    step_skip "llama.cpp bin dir not found"
  fi
fi

# 2: source
if has_selected 2; then
  if [[ -d "$SRC_DIR" ]]; then
    rm -rf "$SRC_DIR"
    step_ok "Removed source"
  else
    step_skip "Source dir not found"
  fi
fi

# 3: configs
if has_selected 3; then
  if [[ -d "$CONFIG_DIR" ]]; then
    rm -rf "$CONFIG_DIR"
    step_ok "Removed configs"
  else
    step_skip "Configs dir not found"
  fi
fi

# 4: logs
if has_selected 4; then
  if [[ -d "$LOGS_DIR" ]]; then
    rm -rf "$LOGS_DIR"
    step_ok "Removed logs"
  else
    step_skip "Logs dir not found"
  fi
fi

# 5: models
if has_selected 5; then
  if [[ -d "$MODELS_DIR" ]]; then
    rm -rf "$MODELS_DIR"
    step_ok "Removed downloaded models"
  else
    step_skip "Models dir not found"
  fi
else
  if [[ -d "$MODELS_DIR" ]]; then
    step_skip "Downloaded models kept at $MODELS_DIR"
  fi
fi

# --- clean PATH from shell rc (only if binary was removed) ---
if has_selected 0; then
  for rc in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_profile"; do
    if [[ -f "$rc" ]] && grep -q "$PREFIX" "$rc" 2>/dev/null; then
      grep -v "export PATH=\"$PREFIX" "$rc" > "${rc}.tmp" && mv "${rc}.tmp" "$rc"
      step_ok "Cleaned PATH from $(basename "$rc")"
    fi
  done
fi

# --- remove home dir if empty ---
if [[ -d "$LLMCONFIG_HOME" ]]; then
  remaining=$(ls -A "$LLMCONFIG_HOME" 2>/dev/null | wc -l | xargs)
  if (( remaining == 0 )); then
    rmdir "$LLMCONFIG_HOME"
    step_ok "Removed ~/.llmconfig"
  fi
fi

echo ""
printf "${GREEN}${BOLD}  Done.${RESET}\n\n"
