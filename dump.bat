@echo off
rem Double-click to print raw HID Input Reports as hex.
rem Use this to discover scratch_axis_byte_index and buttons_byte_range
rem before filling them into config.toml.

cd /d "%~dp0"
title scratch-bridge --dump

if not exist scratch-bridge.exe (
    echo scratch-bridge.exe not found next to this batch file.
    pause
    exit /b 1
)
if not exist config.toml (
    echo config.toml not found. Copy config.example.toml to config.toml and edit at least vid/pid.
    pause
    exit /b 1
)

scratch-bridge.exe --config config.toml --dump
set EXIT=%ERRORLEVEL%

echo.
echo scratch-bridge --dump exited with code %EXIT%. Press any key to close.
pause >nul
exit /b %EXIT%
