# ==============================================================================
# scripts/uninstall_anx.ps1
# Windows Systematic Uninstaller for ANX Platform.
# Cleans up compiled binaries, user PATH registrations, cache, and state.
# ==============================================================================

[CmdletBinding()]
param(
    [switch]$Plan,
    [switch]$DryRun,
    [switch]$PurgeConfigs,
    [switch]$Yes
)

$ErrorActionPreference = "Stop"

$IsPlan = $Plan -or $DryRun
$ProgramsDir = Join-Path $env:LOCALAPPDATA "Programs\anx"
$StateDir    = Join-Path $env:LOCALAPPDATA "anx"
$ConfigDir   = Join-Path $env:APPDATA "anx"

function Write-Step([string]$msg)    { Write-Host "  [STEP] $msg" -ForegroundColor White }
function Write-Success([string]$msg) { Write-Host "  [OK]   $msg" -ForegroundColor Green }
function Write-WarnMsg([string]$msg) { Write-Host "  [WARN] $msg" -ForegroundColor Yellow }
function Write-ErrMsg([string]$msg)  { Write-Host "  [ERR]  $msg" -ForegroundColor Red }

Write-Host "==============================================================================" -ForegroundColor White
Write-Host "  ANX Platform Uninstaller (Windows)" -ForegroundColor White
Write-Host "==============================================================================" -ForegroundColor White

$Targets = @()
if (Test-Path $ProgramsDir) { $Targets += $ProgramsDir }
if (Test-Path $StateDir)    { $Targets += $StateDir }
if ($PurgeConfigs -and (Test-Path $ConfigDir)) { $Targets += $ConfigDir }

if ($IsPlan) {
    Write-Host "`nUninstallation Plan (Dry-Run Preview):" -ForegroundColor Cyan
    Write-Host "  • Binaries to purge:    $ProgramsDir (anx.exe, aitools.exe, aliases)"
    Write-Host "  • State & tools cache:  $StateDir"
    if ($PurgeConfigs) {
        Write-Host "  • User configurations:  $ConfigDir (purge requested)"
    } else {
        Write-Host "  • User configurations:  $ConfigDir (preserved; pass -PurgeConfigs to delete)"
    }
    Write-Host "  • User PATH cleanup:    Remove '$ProgramsDir\bin' from environment"
    Write-Host "`nDry-run complete. Run without -Plan / -DryRun to execute." -ForegroundColor Yellow
    exit 0
}

if (-not $Yes) {
    $title = "Confirm ANX Teardown"
    $message = "Are you sure you want to uninstall ANX and remove binaries from your machine?"
    Write-Host "$message (y/N): " -NoNewline -ForegroundColor Yellow
    $reply = Read-Host
    if ($reply -notmatch "^[yY](es)?$") {
        Write-Host "Uninstallation aborted by user." -ForegroundColor Gray
        exit 0
    }
}

# 1. PATH cleanup
Write-Step "Cleaning User PATH environment variable..."
try {
    $binPath = Join-Path $ProgramsDir "bin"
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -like "*$binPath*") {
        $parts = $userPath -split ";" | Where-Object { $_ -and $_ -ne $binPath }
        $newUserPath = $parts -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        $env:Path = ($env:Path -split ";" | Where-Object { $_ -and $_ -ne $binPath }) -join ";"
        Write-Success "Removed $binPath from User PATH."
    }
} catch {
    Write-WarnMsg "Could not update User PATH: $_"
}

# 2. Delete binaries & programs directory
if (Test-Path $ProgramsDir) {
    Write-Step "Purging ANX programs and binaries ($ProgramsDir)..."
    Remove-Item $ProgramsDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Success "Purged $ProgramsDir."
}

# 3. Delete state, logs and decoupled tools
if (Test-Path $StateDir) {
    Write-Step "Purging ANX local state and cache ($StateDir)..."
    Remove-Item $StateDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Success "Purged $StateDir."
}

# 4. Optional config purge
if ($PurgeConfigs -and (Test-Path $ConfigDir)) {
    Write-Step "Purging user configuration directory ($ConfigDir)..."
    Remove-Item $ConfigDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Success "Purged $ConfigDir."
}

Write-Host "`n"
Write-Success "ANX platform uninstalled successfully from this machine."
