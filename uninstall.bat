@echo off
REM ==============================================================================
REM uninstall.bat - Native Windows Uninstaller Entrypoint for ANX Platform.
REM Delegates to scripts\uninstall_anx.ps1 for clean teardown and PATH cleanup.
REM ==============================================================================

setlocal EnableDelayedExpansion

where powershell.exe >nul 2>nul
if %ERRORLEVEL% equ 0 (
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\uninstall_anx.ps1" %*
    exit /b %ERRORLEVEL%
)

echo [ERROR] PowerShell is required to run the ANX uninstaller on Windows.
exit /b 1
