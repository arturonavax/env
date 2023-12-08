# Windows

```sh
## Download WinGet in https://aka.ms/getwinget
## Set Add-AppxPackage: Import-Module Appx -UseWindowsPowerShell
## Install WinGet: Add-AppxPackage -Path <file>

function InstallWinGet() {
    Import-Module Appx -UseWindowsPowerShell

    $hasPackageManager = Get-AppPackage -name "Microsoft.DesktopAppInstaller"

    if(!$hasPackageManager) {
        Add-AppxPackage -Path "https://aka.ms/Microsoft.VCLibs.x64.14.00.Desktop.appx"

        $releases_url = "https://api.github.com/repos/microsoft/winget-cli/releases/latest"

        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $releases = Invoke-RestMethod -uri "$($releases_url)"
        $latestRelease = $releases.assets | Where { $_.browser_download_url.EndsWith("msixbundle") } | Select -First 1

        Add-AppxPackage -Path $latestRelease.browser_download_url
    }
}

function reloadenv {
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" `
        + [System.Environment]::GetEnvironmentVariable("Path","User")

    . $PROFILE
}

InstallWinGet

# Update all
winget upgrade -h --all

# Install tools
winget install Microsoft.PowerToys
winget install --id Microsoft.Powershell --source winget
winget install Microsoft.PowerShell
winget install 7zip.7zip

winget install --id Git.Git -e --source winget
winget install GitHub.cli

winget install Yarn.Yarn
winget install Microsoft.VisualStudioCode

# Run as Administrator
Start-Process powershell -Verb RunAs -Wait -ArgumentList '-Command "& {
    # Install SSH
    Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0

    # Unlock execution scripts
    Set-ExecutionPolicy RemoteSigned

    # Install more icons
    Set-PSRepository -Name PSGallery -InstallationPolicy Trusted
    Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force
    Install-Module -Name Terminal-Icons -Repository PSGallery
}"'

## Add shell config for Terminal-Icons
New-Item -ItemType File -Path $PROFILE -Force
Add-Content -Path $PROFILE -Value "Import-Module -Name Terminal-Icons"

# Install starship
winget install --id Starship.Starship

## Add shell config for starship
New-Item -ItemType File -Path $PROFILE -Force
Add-Content -Path $PROFILE -Value "Invoke-Expression (&starship init powershell)"

# Install pyenv
Invoke-WebRequest -UseBasicParsing -Uri `
    "https://raw.githubusercontent.com/pyenv-win/pyenv-win/master/pyenv-win/install-pyenv-win.ps1" `
    -OutFile "./install-pyenv-win.ps1"; &"./install-pyenv-win.ps1"

# Install fnm
winget install Schniz.fnm

## Add shell config for fnm
New-Item -ItemType File -Path $PROFILE -Force
Add-Content -Path $PROFILE -Value "fnm env --use-on-cd | Out-String | Invoke-Expression"

# Reload PATH
reloadenv

# Install NodeJS
fnm install --latest

# Insatll pnpm
winget install pnpm

# Install Python
pyenv update
pyenv install 3.12.1
pyenv global 3.12.1

# Reload PATH
reloadenv

python3 -m pip install --upgrade pip
python3 -m pip install --upgrade tk
python3 -m pip install --upgrade setuptools
python3 -m pip install --upgrade wheel

## Install pipx
python3 -m pip install --upgrade pip
python3 -m pip install --force --upgrade --user pipx
python3 -m pipx ensurepath

## Install python tools
pipx install black
pipx install pylint
pipx install beautysh
pipx install cmakelang
pipx install speedtest-cli
pipx install litecli

# Install Golang
winget install Golang.Go

# Install Rust
winget install Rustlang.Rustup

# Install Zoxide
winget install zoxide

## Add shell config for zoxide
New-Item -ItemType File -Path $PROFILE -Force
Add-Content -Path $PROFILE -Value "Invoke-Expression (& { (zoxide init powershell | Out-String) })"

# Install FZF
winget install fzf

# Set WSL2
wsl --set-default-version 2

# Install WSL Ubuntu
wsl --install -d Ubuntu
```
