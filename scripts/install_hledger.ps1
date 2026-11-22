param(
    [Parameter(Mandatory = $true)]
    [string]$BinDir
)

$ErrorActionPreference = 'Stop'
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("anx-hledger-" + [guid]::NewGuid())
$PreviousConfig = $env:EGET_CONFIG

try {
    New-Item -ItemType Directory -Path $TempDir | Out-Null
    $env:EGET_CONFIG = Join-Path $TempDir 'eget.toml'
    New-Item -ItemType File -Path $env:EGET_CONFIG | Out-Null

    # Keep repository shorthand resolution independent of the caller's working directory.
    Push-Location -LiteralPath $TempDir
    try {
        # Upstream publishes x64 Windows binaries, also usable on ARM64 via emulation.
        & eget hledgerorg/hledger --all --system all --asset hledger-windows-x64.zip --to $TempDir
        if ($LASTEXITCODE -ne 0) {
            throw "Eget failed to download hledger (exit $LASTEXITCODE)."
        }
    } finally {
        Pop-Location
    }

    $Binaries = @('hledger-ui.exe', 'hledger-web.exe', 'hledger.exe')
    foreach ($Binary in $Binaries) {
        $File = Get-Item -LiteralPath (Join-Path $TempDir $Binary) -ErrorAction SilentlyContinue
        if (-not $File -or $File.PSIsContainer -or $File.Length -eq 0) {
            throw "hledger release did not contain $Binary"
        }
    }

    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    foreach ($Binary in $Binaries) {
        Copy-Item -LiteralPath (Join-Path $TempDir $Binary) -Destination (Join-Path $BinDir $Binary) -Force
    }
} finally {
    $env:EGET_CONFIG = $PreviousConfig
    if (Test-Path -LiteralPath $TempDir) {
        Remove-Item -LiteralPath $TempDir -Recurse -Force
    }
}
