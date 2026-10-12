# EISEN / DAY — React + Go single EXE

This version keeps the approved React/TypeScript/Vite frontend and embeds its production `dist/` into the Go binary.

## Final behavior

`eisen-day.exe` double-click:

1. Starts the Go backend on localhost only.
2. Uses port 8787, or an available localhost port if 8787 is busy.
3. Serves both `/api/*` and the embedded React production build from the same process.
4. Opens the app URL in the default browser automatically.
5. Node/npm/Go are not required to **run** the finished EXE.

## Build once on Windows

Double-click:

`build-combined-windows.bat`

Or PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\build-combined-windows.ps1
```

Build-time requirements only:

- Node.js + npm
- Go

Output:

`dist\eisen-day.exe`

## Development

You can still run frontend/backend separately while developing:

```powershell
# backend
cd backend
go run ./cmd/eisenday

# frontend (another terminal)
cd frontend
npm install
npm run dev
```

Set `EISEN_DAY_NO_BROWSER=1` if you do not want the Go process to automatically open a browser during development.
