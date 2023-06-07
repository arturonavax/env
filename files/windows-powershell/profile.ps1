# File at $PROFILE
Import-Module -Name Terminal-Icons

Invoke-Expression (&starship init powershell)
Invoke-Expression (& { (zoxide init powershell | Out-String) })

# fnm
fnm env --use-on-cd | Out-String | Invoke-Expression

# Reload PATH
function reloadenv {
	$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" `
		+ [System.Environment]::GetEnvironmentVariable("Path","User")

	. $PROFILE
}

function cpath {
	$tmp_path = Get-Location
	Set-Clipboard -Path $tmp_path
}
