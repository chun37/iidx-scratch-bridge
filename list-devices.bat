@echo off
rem Double-click to list every HID device attached to this PC, with VID/PID.
rem Use the output to fill device.vid and device.pid in config.toml.

cd /d "%~dp0"
title scratch-bridge --list

if not exist scratch-bridge.exe (
    echo scratch-bridge.exe not found next to this batch file.
    pause
    exit /b 1
)

scratch-bridge.exe --list

echo.
echo Press any key to close.
pause >nul
