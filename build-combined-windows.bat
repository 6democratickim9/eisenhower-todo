@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build-combined-windows.ps1"
if errorlevel 1 (
  echo.
  echo Build failed.
  pause
  exit /b 1
)
echo.
pause
