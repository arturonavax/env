# ==============================================================================
# scripts/bootstrap_anx.ps1
# Windows Pre-Flight Analyzer & Golang Bootstrap for ANX Platform.
# ==============================================================================

[CmdletBinding()]
param(
    [string]$BinName = $env:ANX_BIN_NAME,
    [switch]$FromSource = $false,
    [switch]$GoInstall = $false,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
$MinMajor = 1
$MinMinor = 27
$MinPatch = 3

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
        if ($raw -match "go(\d+)\.(\d+)(?:\.(\d+))?") {
            $major = [int]$matches[1]
            $minor = [int]$matches[2]
            $patch = if ($matches[3]) { [int]$matches[3] } else { 0 }
            if ($major -gt $MinMajor) { return $true }
            if ($major -eq $MinMajor -and $minor -gt $MinMinor) { return $true }
            if ($major -eq $MinMajor -and $minor -eq $MinMinor -and $patch -ge $MinPatch) { return $true }
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

        $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
        $file = $rel.files | Where-Object { $_.os -eq "windows" -and $_.arch -eq $arch -and $_.kind -eq "archive" } | Select-Object -First 1
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

function Try-DownloadReleaseBinary {
    if ($FromSource) { return $false }

    $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
    $archiveName = "anx_windows_$arch.zip"
    $releaseUrl = "https://github.com/arturonavax/env/releases/latest/download/$archiveName"
    $sumsUrl = "https://github.com/arturonavax/env/releases/latest/download/SHA256SUMS"

    Write-Step "Checking for verified pre-compiled ANX release ($archiveName)..."
    try {
        $tempDir = Join-Path $env:TEMP ("anx-release-" + [System.Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
        $zipPath = Join-Path $tempDir $archiveName
        $sumsPath = Join-Path $tempDir "SHA256SUMS"

        Invoke-WebRequest -Uri $releaseUrl -OutFile $zipPath -UseBasicParsing -TimeoutSec 30
        Invoke-WebRequest -Uri $sumsUrl -OutFile $sumsPath -UseBasicParsing -TimeoutSec 15

        if ((Test-Path $zipPath) -and (Test-Path $sumsPath)) {
            $sumsContent = Get-Content -Path $sumsPath -Raw
            $match = [regex]::Match($sumsContent, "([a-fA-F0-9]{64})\s+\*?" + [regex]::Escape($archiveName))
            if ($match.Success) {
                $expectedHash = $match.Groups[1].Value.ToLower()
                $actualHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
                if ($actualHash -ne $expectedHash) {
                    Write-WarnMsg "Release checksum mismatch! Expected $expectedHash, got $actualHash"
                    Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
                    return $false
                }
                Write-Success "Release checksum verified: $actualHash"
            }

            $extractDir = Join-Path $tempDir "extract"
            Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
            $anxStaged = Join-Path $extractDir "anx.exe"
            if (Test-Path $anxStaged) {
                # Self test
                & $anxStaged version | Out-Null
                $targetExe = Join-Path $DestDir "anx.exe"
                Move-Item -Path $anxStaged -Destination $targetExe -Force
                Copy-Item -Path $targetExe -Destination (Join-Path $DestDir "safe-suspend.exe") -Force
                Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
                Write-Success "Deployed verified release binary to $targetExe"
                return $true
            }
        }
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    } catch {
        # Fall back to compilation
    }
    return $false
}

function Try-GoInstall {
    if (Get-Command "go" -ErrorAction SilentlyContinue) {
        Write-Step "Installing ANX via 'go install github.com/arturonavax/env/cmd/anx@latest'..."
        try {
            & go install -trimpath github.com/arturonavax/env/cmd/anx@latest
            $gobin = (& go env GOBIN)
            if (-not $gobin) {
                $gobin = Join-Path (& go env GOPATH) "bin"
            }
            $built = Join-Path $gobin "anx.exe"
            if (Test-Path $built) {
                $targetExe = Join-Path $DestDir "anx.exe"
                Copy-Item -Path $built -Destination $targetExe -Force
                Copy-Item -Path $targetExe -Destination (Join-Path $DestDir "safe-suspend.exe") -Force
                Write-Success "Installed via 'go install' and deployed to $targetExe"
                return $true
            }
        } catch { }
    }
    return $false
}

Write-Host "==============================================================================" -ForegroundColor White
Write-Host "  ANX Platform Bootstrap (Windows)" -ForegroundColor White
Write-Host "==============================================================================" -ForegroundColor White

if ($GoInstall) {
    if (Try-GoInstall) {
        # Proceed to dispatch
    } else {
        Write-ErrMsg "Failed to install via 'go install'."
        exit 1
    }
} elseif (-not $FromSource) {
    if (Try-DownloadReleaseBinary) {
        # Verified release deployed
    } elseif (Try-GoInstall) {
        # Installed via go install
    }
}

$anxInstalled = Join-Path $DestDir "anx.exe"
if (-not (Test-Path $anxInstalled)) {
    if (-not (Test-GoVersion)) {
        Write-WarnMsg "Go (>= $MinMajor.$MinMinor.$MinPatch) not detected. Initiating automated bootstrap..."
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
        $safeSuspendExe = Join-Path $DestDir "safe-suspend.exe"

        & go build -ldflags "-s -w -X main.Version=$ver -X main.Commit=$commit -X main.Date=$date" -o $anxExe .\cmd\anx
        Copy-Item $anxExe $safeSuspendExe -Force
        Remove-Item (Join-Path $DestDir "aitools.exe") -ErrorAction SilentlyContinue
        Write-Success "Built $anxExe (alias: safe-suspend.exe)"

        if ($BinName -and $BinName -ne "anx") {
            $customExe = Join-Path $DestDir "$BinName.exe"
            Copy-Item $anxExe $customExe -Force
            Write-Success "Created custom binary alias: $customExe -> $anxExe"
        }
    } finally {
        Pop-Location
    }
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
    $aliasHint = ""
    if ($BinName -and $BinName -ne "anx") {
        $aliasHint = " (or '$BinName')"
    }
    Write-Success "Bootstrap complete! You can now run 'anx'$aliasHint or 'anx wizard'."
}
