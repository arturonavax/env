# install_fonts.ps1 - Idempotent per-user Monaspace Nerd Font (Mono) installer for Windows.
# Installs into %LOCALAPPDATA%\Microsoft\Windows\Fonts and registers in HKCU (no admin required).

$ErrorActionPreference = "Stop"

$FontVersion = if ($env:NERD_FONTS_VERSION) { $env:NERD_FONTS_VERSION } else { "v3.5.1" }
$DownloadUrl = if ($env:ANX_NERD_FONTS_URL) { $env:ANX_NERD_FONTS_URL } else { "https://github.com/ryanoasis/nerd-fonts/releases/download/$FontVersion/Monaspace.zip" }
$ReadyMarker = Join-Path $HOME ".local\share\anx\fonts-monaspace.ready"
$FontsDir = Join-Path $env:LOCALAPPDATA "Microsoft\Windows\Fonts"
$RegPath = "HKCU:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts"

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("anx-fonts-" + [System.Guid]::NewGuid().ToString("N"))

try {
    New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null
    $Archive = Join-Path $TmpDir "Monaspace.zip"
    $ExtractDir = Join-Path $TmpDir "extract"

    Write-Host "[fonts] Downloading Monaspace Nerd Fonts ($FontVersion)..."
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $Archive -UseBasicParsing

    Write-Host "[fonts] Extracting archive..."
    Expand-Archive -Path $Archive -DestinationPath $ExtractDir -Force

    $FontFiles = @()
    if ($env:ANX_NERD_FONTS_ALL -ne "1") {
        $FontFiles = Get-ChildItem -Path $ExtractDir -Recurse -File | Where-Object {
            ($_.Extension -ieq ".otf" -or $_.Extension -ieq ".ttf") -and $_.Name -ilike "*NerdFontMono*"
        }
    }
    if ($FontFiles.Count -eq 0) {
        $FontFiles = Get-ChildItem -Path $ExtractDir -Recurse -File | Where-Object {
            $_.Extension -ieq ".otf" -or $_.Extension -ieq ".ttf"
        }
    }
    if ($FontFiles.Count -eq 0) {
        throw "No .otf or .ttf font files found in $DownloadUrl"
    }

    New-Item -ItemType Directory -Path $FontsDir -Force | Out-Null
    if (-not (Test-Path $RegPath)) {
        New-Item -Path $RegPath -Force | Out-Null
    }

    foreach ($File in $FontFiles) {
        $DestPath = Join-Path $FontsDir $File.Name
        Copy-Item -Path $File.FullName -Destination $DestPath -Force
        $Suffix = if ($File.Extension -ieq ".otf") { "(OpenType)" } else { "(TrueType)" }
        $RegName = "$($File.BaseName) $Suffix"
        New-ItemProperty -Path $RegPath -Name $RegName -Value $DestPath -PropertyType String -Force | Out-Null
    }

    $MarkerDir = Split-Path -Parent $ReadyMarker
    New-Item -ItemType Directory -Path $MarkerDir -Force | Out-Null
    Set-Content -Path $ReadyMarker -Value "version=$FontVersion`ncount=$($FontFiles.Count)`ndir=$FontsDir" -Encoding UTF8
    Write-Host "[fonts] Installed $($FontFiles.Count) Monaspace Nerd Font Mono files into $FontsDir."
} finally {
    if (Test-Path $TmpDir) {
        Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
