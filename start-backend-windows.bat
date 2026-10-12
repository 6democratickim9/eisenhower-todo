@echo off
setlocal
cd /d "%~dp0"
set "EISEN_DAY_ADDR=127.0.0.1:8787"
set "EISEN_DAY_STRICT_ADDR=1"
echo Starting EISEN / DAY Go backend on http://127.0.0.1:8787
bin\eisen-day-backend-windows-amd64.exe
if errorlevel 1 (
  echo.
  echo Backend could not start. If port 8787 is already in use, close the older EISEN / DAY backend first.
  pause
)
