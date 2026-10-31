# ==============================================================================
# install.ps1
# Primary Windows entry point and remote bootstrap for ANX Platform.
# Usage:
#   irm https://sh.arturonavax.dev | iex
#   irm https://sh.arturonavax.dev/install.ps1 | iex
# ==============================================================================

[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

$ErrorActionPreference = "Stop"

function Write-Info([string]$msg) {
    Write-Host "  [INFO] $msg" -ForegroundColor Cyan
}

# 1. Check if running inside an existing local repository clone
$RepoRoot = $null
$cur = (Get-Location).Path
while ($cur -and (Test-Path $cur)) {
    $manifestPath = Join-Path $cur "manifests\configs.yaml"
    $bootstrapPath = Join-Path $cur "scripts\bootstrap_anx.ps1"
    if ((Test-Path $manifestPath) -and (Test-Path $bootstrapPath)) {
        $RepoRoot = $cur
        break
    }
    $parent = Split-Path -Parent $cur
    if ($parent -eq $cur) { break }
    $cur = $parent
}

# 2. Remote Bootstrap: clone repository if not already present
if (-not $RepoRoot) {
    $TargetRepo = if ($env:ANX_REPO_DIR) { $env:ANX_REPO_DIR } else { Join-Path $env:LOCALAPPDATA "anx\repo" }
    Write-Info "Remote execution detected: bootstrapping repository into $TargetRepo..."

    if (-not (Test-Path (Split-Path -Parent $TargetRepo))) {
        New-Item -ItemType Directory -Path (Split-Path -Parent $TargetRepo) -Force | Out-Null
    }

    if (Test-Path (Join-Path $TargetRepo ".git")) {
        try {
            git -C $TargetRepo pull --rebase origin main 2>$null
        } catch { }
    } else {
        $gitCmd = Get-Command "git" -ErrorAction SilentlyContinue
        $gitUrl = if ($env:ANX_GIT_URL) { $env:ANX_GIT_URL } else { "https://github.com/arturonavax/env.git" }

        if ($gitCmd) {
            git clone --depth 1 $gitUrl $TargetRepo
        } else {
            Write-Info "'git' not detected. Fetching repository ZIP archive from GitHub..."
            $zipUrl = "https://github.com/arturonavax/env/archive/refs/heads/main.zip"
            $tempZip = Join-Path $env:TEMP "anx-repo-main.zip"
            $tempExtract = Join-Path $env:TEMP "anx-repo-extract"

            Invoke-WebRequest -Uri $zipUrl -OutFile $tempZip -UseBasicParsing
            if (Test-Path $tempExtract) { Remove-Item -Path $tempExtract -Recurse -Force }
            Expand-Archive -Path $tempZip -DestinationPath $tempExtract -Force

            $extractedRoot = Join-Path $tempExtract "env-main"
            if (-not (Test-Path $TargetRepo)) {
                New-Item -ItemType Directory -Path $TargetRepo -Force | Out-Null
            }
            Copy-Item -Path "$extractedRoot\*" -Destination $TargetRepo -Recurse -Force
            Remove-Item -Path $tempZip -Force -ErrorAction SilentlyContinue
            Remove-Item -Path $tempExtract -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    $RepoRoot = $TargetRepo
}

# 3. Delegate to scripts\bootstrap_anx.ps1
$bootstrapScript = Join-Path $RepoRoot "scripts\bootstrap_anx.ps1"
if (-not (Test-Path $bootstrapScript)) {
    Write-Error "Bootstrap script not found at: $bootstrapScript"
    exit 1
}

& $bootstrapScript @ForwardArgs
