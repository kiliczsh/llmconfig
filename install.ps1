#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$Prefix = "$env:LOCALAPPDATA\llamaconfig\bin",
    [string]$Version = "",
    [switch]$NoLlama,
    [switch]$NoBackends,
    [switch]$Update,
    [switch]$Help
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$RepoSlug   = "kiliczsh/llamaconfig"
$BinaryName = "llamaconfig"
$MinDiskMB  = 200

if ($Help) {
    @"
Usage: install.ps1 [-Prefix PATH] [-Version vX.Y.Z] [-NoBackends] [-Update]

  -Prefix PATH     Install directory (default: %LOCALAPPDATA%\llamaconfig\bin)
  -Version TAG     Install a specific release tag (default: latest)
  -NoBackends      Skip backend downloads (llama.cpp, sd.cpp, whisper.cpp)
  -Update          Force reinstall even if already present

The script downloads a prebuilt binary from GitHub Releases; no Go or
git toolchain is required on the target machine.

When run from inside an extracted release archive (i.e. llamaconfig.exe
sits next to this script), the download step is skipped and the
adjacent binary is installed directly - useful for offline installs.
"@ | Write-Host
    exit 0
}

# --- helpers ---
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

# Detect whether we're running from inside an already-extracted release
# archive. $PSScriptRoot is empty when piped via `iex`.
$LocalMode = $false
if ($PSScriptRoot -and (Test-Path (Join-Path $PSScriptRoot "$BinaryName.exe")) -and [string]::IsNullOrEmpty($Version)) {
    $LocalMode = $true
}

# Remote: pre-flight, resolve, download, install (+ optional llama)
# Local:  pre-flight, use-bundled, install (+ optional llama)
if ($NoLlama) { $NoBackends = $true }
$TotalSteps = if ($LocalMode) { 3 } else { 4 }
if (-not $NoBackends) { $TotalSteps++ }

# ============================================================
# [1] Pre-flight
# ============================================================
step 1 $TotalSteps "Pre-flight checks"

$archRaw = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($archRaw) {
    "AMD64|x86_64" { $arch = "amd64"; break }
    "ARM64"        { $arch = "arm64"; break }
    default        { die "Unsupported arch: $archRaw" }
}
ok "OS: windows/$arch"

$drive = (Split-Path $env:USERPROFILE -Qualifier)
$disk  = Get-PSDrive ($drive.TrimEnd(':'))
$freeMB = [math]::Floor($disk.Free / 1MB)
if ($freeMB -lt $MinDiskMB) { die "Not enough disk space (need ${MinDiskMB}MB, have ${freeMB}MB)" }
ok "Disk: ${freeMB}MB available"

# Local mode only needs internet for the optional llama.cpp step.
if (-not $LocalMode -or -not $NoLlama) {
    try {
        $null = Invoke-WebRequest -Uri "https://github.com" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        ok "Internet: reachable"
    } catch {
        if ($LocalMode) {
            warn "No internet - forcing -NoLlama for this run"
            $NoLlama = $true
            $TotalSteps--
        } else {
            die "No internet connection"
        }
    }
}

# $srcDir is where Install reads binaries from: the extracted archive
# in local mode, a fresh tmpdir filled by the download step otherwise.
$srcDir  = $null
$cleanup = $null

try {
    if ($LocalMode) {
        # ============================================================
        # [2] Use bundled binary
        # ============================================================
        step 2 $TotalSteps "Using bundled binary"
        $srcDir = $PSScriptRoot
        $bundledBin = Join-Path $srcDir "$BinaryName.exe"
        try {
            $versionNoV = [string](& $bundledBin version) -replace "llamaconfig ","" -replace " .*",""
        } catch {
            $versionNoV = "bundled"
        }
        ok "Version: $versionNoV (from $srcDir)"

        if (-not $Update) {
            $existing = Get-Command $BinaryName -ErrorAction SilentlyContinue
            if ($existing) {
                try {
                    $installed = [string](& $existing.Source version) -replace "llamaconfig ","" -replace " .*",""
                } catch {
                    $installed = ""
                }
                if ($installed -eq $versionNoV) {
                    ok "$BinaryName $installed is already installed (use -Update to reinstall)"
                    if ($NoBackends) { exit 0 }
                }
            }
        }
    } else {
        # ============================================================
        # [2] Resolve release
        # ============================================================
        step 2 $TotalSteps "Resolving release"

        if ([string]::IsNullOrEmpty($Version)) {
            info "Fetching latest release tag..."
            try {
                $release = Invoke-RestMethod "https://api.github.com/repos/$RepoSlug/releases/latest" -UseBasicParsing
                $Version = $release.tag_name
            } catch {
                die "Could not determine latest release (API rate-limited? Try -Version vX.Y.Z)"
            }
        }
        $versionNoV = $Version -replace "^v",""
        ok "Version: $Version"

        if (-not $Update) {
            $existing = Get-Command $BinaryName -ErrorAction SilentlyContinue
            if ($existing) {
                try {
                    $installed = [string](& $existing.Source version) -replace "llamaconfig ","" -replace " .*",""
                } catch {
                    $installed = ""
                }
                if ($installed -eq $versionNoV) {
                    ok "$BinaryName $installed is already installed (use -Update to reinstall)"
                    if ($NoBackends) { exit 0 }
                }
            }
        }

        $archive     = "llamaconfig-$versionNoV-windows-$arch.zip"
        $archiveUrl  = "https://github.com/$RepoSlug/releases/download/$Version/$archive"
        $checksumUrl = "https://github.com/$RepoSlug/releases/download/$Version/checksums.txt"

        # ============================================================
        # [3] Download & verify
        # ============================================================
        step 3 $TotalSteps "Downloading binary"

        $srcDir  = New-Item -ItemType Directory -Path ([System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "llamaconfig-install-$(Get-Random)")) -Force
        $cleanup = $srcDir

        $archivePath = Join-Path $srcDir $archive
        info "Downloading $archive..."
        try {
            Invoke-WebRequest -Uri $archiveUrl -OutFile $archivePath -UseBasicParsing -ErrorAction Stop
        } catch {
            die "Download failed: $archiveUrl"
        }
        $sizeMB = [math]::Round((Get-Item $archivePath).Length / 1MB, 1)
        ok "Downloaded ${sizeMB}MB"

        $checksumPath = Join-Path $srcDir "checksums.txt"
        try {
            Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath -UseBasicParsing -ErrorAction Stop
            $expectedLine = (Get-Content $checksumPath | Where-Object { $_ -match "\s+$([regex]::Escape($archive))$" }) | Select-Object -First 1
            if ($expectedLine) {
                $expected = ($expectedLine -split '\s+')[0]
                $actual = (Get-FileHash -Algorithm SHA256 $archivePath).Hash.ToLower()
                if ($actual -ne $expected.ToLower()) {
                    die "Checksum mismatch (expected $expected, got $actual)"
                }
                ok "Checksum verified"
            }
        } catch {
            warn "No checksum file - skipping verification"
        }

        Expand-Archive -Path $archivePath -DestinationPath $srcDir -Force
        if (-not (Test-Path (Join-Path $srcDir "$BinaryName.exe"))) {
            die "Archive did not contain $BinaryName.exe"
        }
    }

    # ============================================================
    # Install binary
    # ============================================================
    step $(if ($LocalMode) { 3 } else { 4 }) $TotalSteps "Installing binary"

    New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
    $dest = Join-Path $Prefix "$BinaryName.exe"
    Copy-Item (Join-Path $srcDir "$BinaryName.exe") $dest -Force
    ok "Installed to $dest"

    # Prefer the bundled llmc.exe from the archive; fall back to a .cmd
    # wrapper for older releases that don't include it.
    $extractedLlmc = Join-Path $srcDir "llmc.exe"
    if (Test-Path $extractedLlmc) {
        $llmcDest = Join-Path $Prefix "llmc.exe"
        Copy-Item $extractedLlmc $llmcDest -Force
        ok "Installed llmc to $llmcDest"
    } else {
        $llmcCmd = Join-Path $Prefix "llmc.cmd"
        Set-Content -Path $llmcCmd -Value "@echo off`r`n`"$dest`" %*" -Encoding ASCII
        ok "Alias: llmc -> llamaconfig  ($llmcCmd)"
    }

    # PATH
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($userPath -notlike "*$Prefix*") {
        [Environment]::SetEnvironmentVariable("PATH", "$Prefix;$userPath", "User")
        $env:PATH = "$Prefix;$env:PATH"
        ok "Added $Prefix to user PATH (restart terminal to apply)"
    } else {
        ok "PATH already contains $Prefix"
    }

    # Smoke test
    try { $installedVersion = & $dest version } catch { $installedVersion = "installed" }
    ok $installedVersion

    # ============================================================
    # Install backends (llama.cpp, stable-diffusion.cpp, whisper.cpp)
    # ============================================================
    if (-not $NoBackends) {
        step $TotalSteps $TotalSteps "Installing backends"

        # llama.cpp
        try { $llamaPath = & $dest llama --path } catch { $llamaPath = $null }
        if ($llamaPath -and (Test-Path $llamaPath) -and -not $Update) {
            try { $llamaVer = (& $dest llama --version | Select-String "version:") -replace ".*version: ","" } catch { $llamaVer = $null }
            ok "llama.cpp already installed: $(if ($llamaVer) { $llamaVer } else { 'unknown version' })"
        } else {
            info "Downloading llama.cpp..."
            & $dest install llama
            try { $llamaVer = (& $dest llama --version | Select-String "version:") -replace ".*version: ","" } catch { $llamaVer = $null }
            ok "llama.cpp: $(if ($llamaVer) { $llamaVer } else { 'installed' })"
        }

        # stable-diffusion.cpp
        try { $sdPath = & $dest sd --path } catch { $sdPath = $null }
        if ($sdPath -and (Test-Path $sdPath) -and -not $Update) {
            try { $sdVer = [string](& $dest sd --version) -replace ".*commit ","" -replace "\s.*","" } catch { $sdVer = $null }
            ok "stable-diffusion.cpp already installed: $(if ($sdVer) { "commit $sdVer" } else { 'installed' })"
        } else {
            info "Downloading stable-diffusion.cpp..."
            & $dest install sd
            ok "stable-diffusion.cpp: installed"
        }

        # whisper.cpp
        try { $whisperPath = & $dest whisper --path } catch { $whisperPath = $null }
        if ($whisperPath -and (Test-Path $whisperPath) -and -not $Update) {
            ok "whisper.cpp already installed"
        } else {
            info "Downloading whisper.cpp..."
            & $dest install whisper
            ok "whisper.cpp: installed"
        }
    }
} finally {
    if ($cleanup) {
        Remove-Item $cleanup -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# ============================================================
# Done
# ============================================================
Write-Host ""
Write-Host "  Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "  Run: " -NoNewline; Write-Host "llmc.exe init --template gemma" -ForegroundColor Cyan
Write-Host "       " -NoNewline; Write-Host "llmc.exe up <model-name>" -ForegroundColor Cyan
Write-Host ""
