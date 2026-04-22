#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$Prefix = "$env:LOCALAPPDATA\llamaconfig\bin",
    [switch]$All,
    [switch]$KeepCache,
    [switch]$Help
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$BinaryName    = "llamaconfig"
$LlamaHome     = Join-Path $env:USERPROFILE ".llamaconfig"
$SrcDir        = Join-Path $LlamaHome "src"        # legacy (old source-install)
$BinDir        = Join-Path $LlamaHome "bin"        # llama.cpp, sd, whisper binaries
$ConfigDir     = Join-Path $LlamaHome "configs"
$LogsDir       = Join-Path $LlamaHome "logs"
$CacheDir      = Join-Path $LlamaHome "cache"

if ($Help) {
    @"
Usage: uninstall.ps1 [-Prefix PATH] [-All] [-KeepCache]

  -Prefix PATH    Install directory to remove from
                  (default: %LOCALAPPDATA%\llamaconfig\bin)
  -All            Remove everything without prompting (binary, configs,
                  logs, llama.cpp, cache). Equivalent to answering "yes"
                  to every prompt.
  -KeepCache      Skip the model cache prompt/removal (useful with -All).

Interactive by default: asks yes/no per category.
"@ | Write-Host
    exit 0
}

function ok   { param($msg) Write-Host "  [OK] $msg" -ForegroundColor Green }
function skip { param($msg) Write-Host "  [-]  $msg" -ForegroundColor DarkGray }
function warn { param($msg) Write-Host "  [!]  $msg" -ForegroundColor Yellow }

function Get-SizeLabel {
    param($path)
    if (-not (Test-Path $path)) { return "-" }
    try {
        $bytes = (Get-ChildItem $path -Recurse -Force -ErrorAction SilentlyContinue | Measure-Object Length -Sum).Sum
        if (-not $bytes) { return "0B" }
        if ($bytes -gt 1GB) { return ("{0:N1}GB" -f ($bytes / 1GB)) }
        if ($bytes -gt 1MB) { return ("{0:N1}MB" -f ($bytes / 1MB)) }
        if ($bytes -gt 1KB) { return ("{0:N1}KB" -f ($bytes / 1KB)) }
        return "${bytes}B"
    } catch {
        return "?"
    }
}

function Ask {
    param(
        [string]$Label,
        [string]$Path,
        [switch]$DefaultNo
    )
    if ($All) { return $true }
    $size = Get-SizeLabel $Path
    $suffix = if ($DefaultNo) { "[y/N]" } else { "[Y/n]" }
    $answer = Read-Host "  Remove $Label ($size)? $suffix"
    if ($DefaultNo) {
        return ($answer -match "^[yY]")
    } else {
        return (-not ($answer -match "^[nN]"))
    }
}

Write-Host ""
Write-Host @"
  _   _       _           _        _ _
 | | | |_ __ (_)_ __  ___| |_ __ _| | |
 | | | | '_ \| | '_ \/ __| __/ _' | | |
 | |_| | | | | | | | \__ \ || (_| | | |
  \___/|_| |_|_|_| |_|___/\__\__,_|_|_|
"@ -ForegroundColor Red
Write-Host ""

$binaryPath = Join-Path $Prefix "$BinaryName.exe"
$lcPath     = Join-Path $Prefix "lc.cmd"

# Stop running models before we yank the binary
if (Test-Path $binaryPath) {
    try { & $binaryPath down --all 2>$null } catch { }
}

# --- Binary ---
if (Test-Path $binaryPath) {
    if (Ask -Label "llamaconfig binary" -Path $binaryPath) {
        Remove-Item $binaryPath -Force
        ok "Removed $binaryPath"
        if (Test-Path $lcPath) { Remove-Item $lcPath -Force; ok "Removed lc alias" }

        # Clean PATH
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -like "*$Prefix*") {
            $newPath = ($userPath -split ";" | Where-Object { $_ -ne $Prefix }) -join ";"
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
            ok "Removed $Prefix from user PATH"
        }
    } else {
        skip "Kept binary"
    }
} else {
    skip "Binary not found at $binaryPath"
}

# --- Legacy source dir ---
if (Test-Path $SrcDir) {
    if (Ask -Label "legacy source dir" -Path $SrcDir) {
        Remove-Item $SrcDir -Recurse -Force
        ok "Removed $SrcDir"
    } else {
        skip "Kept source dir"
    }
}

# --- llama.cpp / sd / whisper binaries ---
if (Test-Path $BinDir) {
    if (Ask -Label "backend binaries (llama.cpp, sd, whisper)" -Path $BinDir) {
        Remove-Item $BinDir -Recurse -Force
        ok "Removed $BinDir"
    } else {
        skip "Kept backend binaries"
    }
}

# --- Configs ---
if (Test-Path $ConfigDir) {
    if (Ask -Label "model configs" -Path $ConfigDir) {
        Remove-Item $ConfigDir -Recurse -Force
        ok "Removed $ConfigDir"
    } else {
        skip "Kept configs"
    }
}

# --- Logs ---
if (Test-Path $LogsDir) {
    if (Ask -Label "logs" -Path $LogsDir) {
        Remove-Item $LogsDir -Recurse -Force
        ok "Removed $LogsDir"
    } else {
        skip "Kept logs"
    }
}

# --- Cache (default no) ---
if ((Test-Path $CacheDir) -and -not $KeepCache) {
    if (Ask -Label "model cache (GGUF files)" -Path $CacheDir -DefaultNo) {
        Remove-Item $CacheDir -Recurse -Force
        ok "Removed $CacheDir"
    } else {
        skip "Kept model cache at $CacheDir"
    }
} elseif (Test-Path $CacheDir) {
    skip "Kept model cache at $CacheDir (--KeepCache)"
}

# --- Remove home dir if empty ---
if (Test-Path $LlamaHome) {
    $remaining = @(Get-ChildItem $LlamaHome -Force -ErrorAction SilentlyContinue)
    if ($remaining.Count -eq 0) {
        Remove-Item $LlamaHome -Force
        ok "Removed $LlamaHome"
    }
}

# --- Remove empty prefix dir ---
if (Test-Path $Prefix) {
    $remaining = @(Get-ChildItem $Prefix -Force -ErrorAction SilentlyContinue)
    if ($remaining.Count -eq 0) {
        Remove-Item $Prefix -Force
        ok "Removed empty $Prefix"
    }
}

Write-Host ""
Write-Host "  Done." -ForegroundColor Green
Write-Host ""
