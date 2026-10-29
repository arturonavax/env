# ==============================================================================
# scripts/bootstrap_anx.ps1
# Windows Pre-Flight Analyzer & Golang Bootstrap for ANX Platform.
# ==============================================================================

[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$MinMajor = 1
$MinMinor = 24

$DestDir = Join-Path $env:LOCALAPPDATA "Programs\anx\bin"
if (-not (Test-Path $DestDir)) {
    New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
}

function Write-Step([string]$msg) {
    Write-Host "  [STEP] $msg" -ForegroundColor White
}
function Write-Success([string]$msg) {
    Write-Host "  [OK]   $msg" -ForegroundColor Green
}
function Write-WarnMsg([string]$msg) {
    Write-Host "  [WARN] $msg" -ForegroundColor Yellow
}
function Write-ErrMsg([string]$msg) {
    Write-Host "  [ERR]  $msg" -ForegroundColor Red
}

function Test-GoVersion {
    $goCmd = Get-Command "go" -ErrorAction SilentlyContinue
    if (-not $goCmd) {
        $candidate = Join-Path $env:LOCALAPPDATA "Programs\go\bin\go.exe"
        if (Test-Path $candidate) {
            $env:Path = (Join-Path $env:LOCALAPPDATA "Programs\go\bin") + ";" + $env:Path
            $goCmd = Get-Command "go" -ErrorAction SilentlyContinue
        }
    }
    if (-not $goCmd) {
        return $false
    }

    try {
        $raw = (& go version)
        if ($raw -match "go(\d+)\.(\d+)") {
            $major = [int]$matches[1]
            $minor = [int]$matches[2]
            if ($major -gt $MinMajor -or ($major -eq $MinMajor -and $minor -ge $MinMinor)) {
                return $true
            }
        }
    } catch {
        return $false
    }
    return $false
}

function Install-GoWindowsArchive {
    Write-Step "Downloading official verified Go release from go.dev..."
    try {
        $json = Invoke-RestMethod -Uri "https://go.dev/dl/?mode=json" -UseBasicParsing
        $rel = $json | Where-Object { $_.stable -eq $true } | Select-Object -First 1
        if (-not $rel) { return $false }

        $file = $rel.files | Where-Object { $_.os -eq "windows" -and $_.arch -eq "amd64" -and $_.kind -eq "archive" } | Select-Object -First 1
        if (-not $file) { return $false }

        $url = "https://go.dev/dl/" + $file.filename
        $expectedHash = $file.sha256
        $tempZip = Join-Path $env:TEMP $file.filename

        Write-Step "Downloading $url..."
        Invoke-WebRequest -Uri $url -OutFile $tempZip -UseBasicParsing

        $actualHash = (Get-FileHash -Path $tempZip -Algorithm SHA256).Hash.ToLower()
        if ($actualHash -ne $expectedHash.ToLower()) {
            Write-ErrMsg "SHA256 mismatch! Expected $expectedHash, got $actualHash"
            Remove-Item $tempZip -Force -ErrorAction SilentlyContinue
            return $false
        }

        $targetGoDir = Join-Path $env:LOCALAPPDATA "Programs\go"
        if (Test-Path $targetGoDir) {
            Remove-Item $targetGoDir -Recurse -Force
        }

        Write-Step "Extracting Go archive to $targetGoDir..."
        Expand-Archive -Path $tempZip -DestinationPath (Join-Path $env:LOCALAPPDATA "Programs") -Force
        Remove-Item $tempZip -Force -ErrorAction SilentlyContinue

        $goBin = Join-Path $targetGoDir "bin"
        $env:Path = "$goBin;" + $env:Path
        [Environment]::SetEnvironmentVariable("Path", "$goBin;" + [Environment]::GetEnvironmentVariable("Path", "User"), "User")
        return $true
    } catch {
        Write-WarnMsg "Failed to install from go.dev: $_"
        return $false
    }
}

function Install-GoWingetOrScoop {
    if (Get-Command "winget" -ErrorAction SilentlyContinue) {
        Write-Step "Attempting Go install via winget..."
        try {
            & winget install --id GoLang.Go -e --silent --accept-package-agreements --accept-source-agreements
            return $true
        } catch { }
    }
    if (Get-Command "scoop" -ErrorAction SilentlyContinue) {
        Write-Step "Attempting Go install via scoop..."
        try {
            & scoop install go
            return $true
        } catch { }
    }
    return $false
}

Write-Host "==============================================================================" -ForegroundColor White
Write-Host "  ANX Platform Bootstrap (Windows)" -ForegroundColor White
Write-Host "==============================================================================" -ForegroundColor White

if (-not (Test-GoVersion)) {
    Write-WarnMsg "Go (>= 1.24) not detected. Initiating automated bootstrap..."
    $installed = Install-GoWindowsArchive
    if (-not $installed) {
        $installed = Install-GoWingetOrScoop
    }
    if (-not (Test-GoVersion)) {
        Write-ErrMsg "Could not bootstrap Go. Please install Go manually from https://go.dev/dl/."
        exit 1
    }
}

$goVer = (& go version)
Write-Success "Go runtime ready: $goVer"

# Build ANX
Write-Step "Building ANX binary for Windows..."
Push-Location $RepoRoot
try {
    $ver = "1.0.0"
    $commit = "dev"
    $date = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd")

    if (Get-Command "git" -ErrorAction SilentlyContinue) {
        $gVer = (& git describe --tags --always 2>$null)
        if ($gVer) { $ver = $gVer }
        $gCommit = (& git rev-parse --short HEAD 2>$null)
        if ($gCommit) { $commit = $gCommit }
    }

    $anxExe = Join-Path $DestDir "anx.exe"
    $aitoolsExe = Join-Path $DestDir "aitools.exe"

    & go build -ldflags "-s -w -X main.Version=$ver -X main.Commit=$commit -X main.Date=$date" -o $anxExe .\cmd\anx
    Copy-Item $anxExe $aitoolsExe -Force
    Write-Success "Built $anxExe and $aitoolsExe"
} finally {
    Pop-Location
}

# Update PATH
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$DestDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$DestDir;$userPath", "User")
    $env:Path = "$DestDir;" + $env:Path
}

if ($ForwardArgs.Count -gt 0) {
    & (Join-Path $DestDir "anx.exe") @ForwardArgs
} else {
    Write-Success "Bootstrap complete! You can now run 'anx' or 'anx wizard'."
}
