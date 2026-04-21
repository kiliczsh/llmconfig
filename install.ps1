#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$Prefix = "$env:LOCALAPPDATA\llamaconfig\bin",
    [switch]$NoLlama,
    [switch]$Update,
    [switch]$Help
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$BinaryName  = "llamaconfig"
$Repo        = "https://github.com/kiliczsh/llamaconfig.git"
$SrcDir      = "$env:USERPROFILE\.llamaconfig\src"
$MinDiskMB   = 600

if ($Help) {
    Write-Host "Usage: install.ps1 [-Prefix PATH] [-NoLlama] [-Update]"
    exit 0
}

# --- colors ---
function ok   { param($msg) Write-Host "  [OK] $msg" -ForegroundColor Green }
function fail { param($msg) Write-Host "  [X]  $msg" -ForegroundColor Red }
function info { param($msg) Write-Host "  [>]  $msg" -ForegroundColor Cyan }
function warn { param($msg) Write-Host "  [!]  $msg" -ForegroundColor Yellow }
function die  { param($msg) fail $msg; exit 1 }

function step {
    param($n, $total, $msg)
    Write-Host ""
    Write-Host "[$n/$total] $msg" -ForegroundColor White
}

# --- banner ---
Write-Host ""
Write-Host @"
  _     _                                       __ _
 | |   | |                                     / _(_)
 | |   | | __ _ _ __ ___   __ _  ___ ___  _ _ | |_ _  __ _
 | |   | |/ _' | '_ ' _ \ / _' |/ __/ _ \| '_ \|  _| |/ _' |
 | |___| | (_| | | | | | | (_| | (_| (_) | | | | | | | (_| |
 |_____|_|\__,_|_| |_| |_|\__,_|\___\___/|_| |_|_| |_|\__, |
                                                         __/ |
                                                        |___/
"@ -ForegroundColor Cyan
Write-Host "  Manage local LLM inference with llama.cpp`n"

$TotalSteps = if ($NoLlama) { 5 } else { 6 }

# ============================================================
# [1] Pre-flight
# ============================================================
step 1 $TotalSteps "Pre-flight checks"

# OS / Arch
$arch = if ([System.Environment]::Is64BitOperatingSystem) { "x64" } else { "x86" }
ok "OS: Windows/$arch"

# Disk
$drive = (Split-Path $env:USERPROFILE -Qualifier)
$disk  = Get-PSDrive ($drive.TrimEnd(':'))
$freeMB = [math]::Floor($disk.Free / 1MB)
if ($freeMB -lt $MinDiskMB) { die "Not enough disk space (need ${MinDiskMB}MB, have ${freeMB}MB)" }
ok "Disk: ${freeMB}MB available"

# Internet
try {
    $null = Invoke-WebRequest -Uri "https://github.com" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    ok "Internet: reachable"
} catch {
    die "No internet connection"
}

# Git
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    die "git not found — install Git: https://git-scm.com/download/win"
}
$gitVer = (git --version) -replace "git version ",""
ok "Git: $gitVer"

# Go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    die "Go not found — install Go: https://go.dev/dl/"
}
$goVer = (go version) -replace "go version go","" -replace " .*",""
ok "Go: $goVer"

# ============================================================
# [2] Clone / Update
# ============================================================
step 2 $TotalSteps "Repository"

New-Item -ItemType Directory -Force -Path (Split-Path $SrcDir) | Out-Null

if (Test-Path "$SrcDir\.git") {
    info "Updating repository..."
    git -C $SrcDir pull --ff-only --quiet
    if ($LASTEXITCODE -ne 0) { die "git pull failed" }
    ok "Repository updated"
} else {
    info "Cloning repository..."
    git clone --depth=1 --quiet $Repo $SrcDir
    if ($LASTEXITCODE -ne 0) { die "git clone failed" }
    ok "Repository cloned to $SrcDir"
}

# ============================================================
# [3] Build
# ============================================================
step 3 $TotalSteps "Building llamaconfig"

info "Compiling..."
Push-Location $SrcDir
try {
    go build -o "$BinaryName.exe" .
    if ($LASTEXITCODE -ne 0) { die "Build failed" }
} finally {
    Pop-Location
}
ok "Build successful"

# ============================================================
# [4] Install binary
# ============================================================
step 4 $TotalSteps "Installing binary"

New-Item -ItemType Directory -Force -Path $Prefix | Out-Null

$dest = "$Prefix\$BinaryName.exe"
Copy-Item "$SrcDir\$BinaryName.exe" $dest -Force
ok "Installed to $dest"

# lc.cmd alias (Windows has no ln -s for exes without admin; .cmd wrapper works everywhere)
$lcCmd = "$Prefix\lc.cmd"
Set-Content -Path $lcCmd -Value "@echo off`r`n`"$dest`" %*" -Encoding ASCII
ok "Alias: lc -> llamaconfig  ($lcCmd)"

# PATH
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$Prefix*") {
    [Environment]::SetEnvironmentVariable("PATH", "$Prefix;$userPath", "User")
    $env:PATH = "$Prefix;$env:PATH"
    ok "Added $Prefix to user PATH (restart terminal to apply)"
} else {
    ok "PATH already contains $Prefix"
}

# ============================================================
# [5] Smoke test
# ============================================================
step 5 $TotalSteps "Smoke test"

$version = & $dest version 2>$null
ok "llamaconfig $version"

$hw = (& $dest hardware 2>$null | Select-String "Selected profile") -replace ".*: ",""
ok "Hardware profile: $(if ($hw) { $hw } else { 'detected' })"

# ============================================================
# [6] Install llama.cpp
# ============================================================
if (-not $NoLlama) {
    step 6 $TotalSteps "Installing llama.cpp"
    $llamaPath = & $dest llama --path 2>$null
    if ($llamaPath -and (Test-Path $llamaPath) -and -not $Update) {
        $llamaVer = (& $dest llama --version 2>$null | Select-String "version:") -replace ".*version: ",""
        ok "llama.cpp already installed: $(if ($llamaVer) { $llamaVer } else { 'unknown version' }) (use -Update to reinstall)"
    } else {
        info "Downloading llama.cpp binary..."
        & $dest llama --install
        $llamaVer = (& $dest llama --version 2>$null | Select-String "version:") -replace ".*version: ",""
        ok "llama.cpp: $(if ($llamaVer) { $llamaVer } else { 'installed' })"
    }
}

# ============================================================
# Done
# ============================================================
Write-Host ""
Write-Host "  Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "  Run: " -NoNewline; Write-Host "lc init --template gemma" -ForegroundColor Cyan
Write-Host "       " -NoNewline; Write-Host "lc up <model-name>" -ForegroundColor Cyan
Write-Host ""
