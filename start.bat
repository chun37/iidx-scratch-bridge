@echo off
rem Double-click to run scratch-bridge against config.toml.
rem Keep this file, scratch-bridge.exe, and config.toml in the same folder.

cd /d "%~dp0"
title scratch-bridge

if not exist scratch-bridge.exe (
    echo scratch-bridge.exe not found next to this batch file.
    pause
    exit /b 1
)
if not exist config.toml (
    echo config.toml not found. Copy config.example.toml to config.toml and edit it.
    pause
    exit /b 1
)

scratch-bridge.exe --config config.toml
set EXIT=%ERRORLEVEL%

echo.
echo scratch-bridge exited with code %EXIT%. Press any key to close.
pause >nul
exit /b %EXIT%
