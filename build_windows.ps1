$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

Write-Host '[1/5] Preparing Python build environment...'
if (-not (Test-Path '.venv-build')) {
    py -3 -m venv .venv-build
}
& .\.venv-build\Scripts\python.exe -m pip install --upgrade pip
& .\.venv-build\Scripts\python.exe -m pip install -r requirements.txt 'pyinstaller>=6.10,<7' 'pywebview>=5,<7'

Write-Host '[2/5] Building React frontend...'
Push-Location frontend
if (Test-Path 'package-lock.json') { npm ci } else { npm install }
npm run build
Pop-Location

Write-Host '[3/5] Running backend tests...'
& .\.venv-build\Scripts\python.exe -m pip install 'pytest>=8,<10'
$env:PYTHONPATH = $Root
& .\.venv-build\Scripts\python.exe -m pytest -q backend\tests

Write-Host '[4/5] Building Windows executable...'
& .\.venv-build\Scripts\pyinstaller.exe --noconfirm --clean EisenDay.spec

Write-Host '[5/5] Creating release-only portable ZIP...'
$ReleaseDir = Join-Path $Root 'release'
New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null
$Zip = Join-Path $ReleaseDir 'EisenDay-windows-portable.zip'
if (Test-Path $Zip) { Remove-Item $Zip -Force }
Compress-Archive -Path (Join-Path $Root 'dist\EisenDay.exe') -DestinationPath $Zip

Write-Host ''
Write-Host 'Done:'
Write-Host '  dist\EisenDay.exe        (build output)'
Write-Host '  release\EisenDay-windows-portable.zip  (user release; contains only EisenDay.exe)'
