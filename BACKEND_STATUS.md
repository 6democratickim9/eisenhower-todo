# Backend implementation status

## Completed

- FastAPI local API matching frontend adapter
- AP create/update/delete/complete/archive restore
- Queue reorder/mode + oldest active AP pinned to Priority 1
- Daily Checkpoint create/update/delete/toggle + disabled protection
- Local JSON persistence, atomic replace, `.bak` recovery
- Cross-platform sidecar file lock
- Runtime data-path switching
- Legacy Go schema migration (`dailyTasks`, `jiraTickets`)
- Legacy `%APPDATA%/EisenhowerTodo` settings migration
- Jira Bearer/Basic auth, API v2/v3, test endpoint
- AP Jira metadata refresh on create/update
- Jira Watch sync, Done removal, completed AP Jira retention
- Jira token separated from Planner state
- Auto-sync loop
- 127.0.0.1-only launcher + dynamic port fallback
- pywebview native-window path for Windows packaged build
- PyInstaller Windows build script/spec
- Frontend production adapter changed to same-origin API

## Verified in this environment

- Python compileall: PASS
- pytest: 7 PASS
- FastAPI HTTP smoke test: PASS
- TS/TSX syntax transpilation check: PASS (15 files)

## Not executable-verified here

- `npm run build`: npm registry DNS/network is unavailable in this execution environment, so React/Vite dependency installation could not be completed here.
- Windows `.exe`: PyInstaller Windows executables must be built on Windows. `build_windows.ps1` is included to perform frontend build, tests, EXE build, and portable ZIP creation on Windows.

## macOS packaging support

- `~/Library/Application Support/EisenDay` 기본 설정/데이터 경로 지원
- POSIX `fcntl` sidecar file lock 사용
- `build_macos.sh` 추가: frontend build -> pytest -> PyInstaller `.app` -> macOS-safe ZIP
- `EisenDay_macos.spec` 추가: pywebview/Cocoa windowed onedir app bundle
- Apple Silicon arm64 / Intel x86_64 native build, optional universal2 target
- Developer ID가 없을 때는 ad-hoc/local build이며 공증은 별도 단계
