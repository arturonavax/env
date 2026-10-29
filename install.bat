@echo off
REM ==============================================================================
REM install.bat - Native Windows Bootstrap Entrypoint for ANX Platform.
REM Delegates to scripts\bootstrap_anx.ps1 with automated Go bootstrap.
REM ==============================================================================

setlocal EnableDelayedExpansion

where powershell.exe >nul 2>nul
if %ERRORLEVEL% equ 0 (
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\bootstrap_anx.ps1" %*
    exit /b %ERRORLEVEL%
)

echo [WARN] PowerShell not detected. Falling back to basic Go build...
where go.exe >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go runtime not found in PATH. Please install Go from https://go.dev/dl/.
    exit /b 1
)

go build -o anx.exe ./cmd/anx
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Failed to compile anx.exe.
    exit /b 1
)

copy /y anx.exe aitools.exe >nul
.\anx.exe %*
endlocal
