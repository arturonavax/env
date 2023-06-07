# Windows

```sh
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
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" `
    + [System.Environment]::GetEnvironmentVariable("Path","User")

# Install NodeJS
fnm install --latest

# Insatll pnpm
winget install pnpm

# Install Python
pyenv install 3.11.3
pyenv global 3.11.3

## Install pipx
python3 -m pip install --upgrade pip
python3 -m pip install --force --upgrade --user pipx
python3 -m pipx ensurepath

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
