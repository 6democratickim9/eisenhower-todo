$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Frontend = Join-Path $Root "frontend"
$Backend = Join-Path $Root "backend"
$EmbedDist = Join-Path $Backend "internal\webui\dist"
$OutDir = Join-Path $Root "dist"

Write-Host "[1/5] Checking tools..." -ForegroundColor Cyan
if (-not (Get-Command node -ErrorAction SilentlyContinue)) { throw "Node.js is required only to BUILD the frontend." }
if (-not (Get-Command npm.cmd -ErrorAction SilentlyContinue))  { throw "npm is required only to BUILD the frontend." }
if (-not (Get-Command go -ErrorAction SilentlyContinue))   { throw "Go is required only to BUILD the final EXE." }

Write-Host "[2/5] Building React/Vite frontend..." -ForegroundColor Cyan
Push-Location $Frontend
try {
  if (Test-Path "package-lock.json") {
    npm.cmd ci
  } else {
    npm.cmd install
  }
  if ($LASTEXITCODE -ne 0) { throw "npm dependency installation failed (exit code $LASTEXITCODE)." }

  npm.cmd run build
  if ($LASTEXITCODE -ne 0) { throw "Frontend build failed (exit code $LASTEXITCODE). Fix the TypeScript/Vite errors above before continuing." }
} finally { Pop-Location }

Write-Host "[3/5] Embedding frontend build into Go..." -ForegroundColor Cyan
$FrontendDist = Join-Path $Frontend "dist"
if (-not (Test-Path $FrontendDist)) { throw "Frontend build did not create frontend\dist. Build aborted before Go embedding." }
if (Test-Path $EmbedDist) { Remove-Item $EmbedDist -Recurse -Force }
New-Item -ItemType Directory -Path $EmbedDist -Force | Out-Null
Copy-Item (Join-Path $FrontendDist "*") $EmbedDist -Recurse -Force

Write-Host "[4/5] Running Go tests..." -ForegroundColor Cyan
Push-Location $Backend
try {
  go test ./...
  if ($LASTEXITCODE -ne 0) { throw "Go tests failed (exit code $LASTEXITCODE)." }
} finally { Pop-Location }

Write-Host "[5/5] Building standalone Windows app..." -ForegroundColor Cyan
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
$Exe = Join-Path $OutDir "eisen-day.exe"
$BackendExe = Join-Path $Root "bin\eisen-day-backend-windows-amd64.exe"
New-Item -ItemType Directory -Path (Split-Path -Parent $BackendExe) -Force | Out-Null
Push-Location $Backend
try {
  go build -trimpath -ldflags "-s -w -H=windowsgui" -o $Exe ./cmd/eisenday
  if ($LASTEXITCODE -ne 0) { throw "Go combined build failed (exit code $LASTEXITCODE)." }

  # Keep the separately runnable backend binary in sync with the same source.
  go build -trimpath -ldflags "-s -w" -o $BackendExe ./cmd/eisenday
  if ($LASTEXITCODE -ne 0) { throw "Go backend build failed (exit code $LASTEXITCODE)." }
} finally { Pop-Location }

Write-Host ""
Write-Host "DONE" -ForegroundColor Green
Write-Host "Standalone app: $Exe"
Write-Host "Separate backend: $BackendExe"
Write-Host "Double-click eisen-day.exe. React UI + Go backend run together; Node/npm/Go are NOT needed after build."
