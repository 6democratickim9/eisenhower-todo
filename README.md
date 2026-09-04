# EISEN / DAY v2

**A local-first personal operations dashboard for people who manage real work, not just to-do lists.**

EISEN / DAY combines an Eisenhower Matrix, execution queue, recurring daily checks, Jira tracking, calendar, tags, and lightweight statistics in one desktop workspace.

The UI is built with **React + TypeScript + Vite** and the local backend runs on **Python + FastAPI**. Packaged builds launch everything together as a single desktop app.

> **No separate server is required for normal use.**  
> On Windows, open `EisenDay.exe`. On macOS, open `EisenDay.app`.

---

## What EISEN / DAY does

EISEN / DAY is designed around three different kinds of work:

### Action Points (AP)

Action Points are normal one-time tasks. Each AP can have:

- title and details
- importance score
- base urgency score
- deadline
- tags
- Jira issue link
- queue position
- creation/update/completion timestamps

The app calculates **Effective Urgency** by combining your base urgency with deadline pressure, then automatically places the AP into the Eisenhower Matrix:

| Quadrant | Meaning |
| --- | --- |
| **Q1** | Urgent + Important |
| **Q2** | Important, not urgent |
| **Q3** | Urgent, less important |
| **Q4** | Lower priority |

Dragging an AP between quadrants updates its priority inputs instead of storing a fake visual-only quadrant.

### Daily Checkpoints

Daily Checkpoints are for things that need to be verified every day, such as:

- checking batch jobs
- reviewing monitoring dashboards
- confirming backups
- checking operational reports
- reviewing Jira progress

A checkpoint can be completed **for today** without disappearing permanently. The next day it becomes pending again automatically because completion is stored by date.

Checkpoints can also be **disabled without being deleted**, so recurring routines can be paused and re-enabled later.

### Jira Watch

Jira Watch keeps track of issues you still care about even when the local AP is complete.

It can store and sync:

- issue key
- summary
- status
- status category
- assignee
- latest comment
- Jira created/updated timestamps
- last sync time
- sync errors

Issues in Jira's **Done** category are removed from the watch list during sync. A completed local AP can still remain visible in Jira Watch if its Jira issue is not done yet.

Jira Issue Keys are clickable links. When a Jira Base URL is configured, `OPS-123` opens `<Jira Base URL>/browse/OPS-123`. EISEN / DAY also normalizes accidentally pasted ticket or REST API URLs back to the Jira base path before generating links.

---

## Main features

### Eisenhower Matrix

The home workspace keeps the core workload in a 2x2 matrix. APs are classified automatically using **Importance** and **Effective Urgency**.

Effective Urgency increases as a deadline gets closer, so an AP can naturally move from Q2 to Q1 as time pressure increases.

### Today's Queue

The queue answers a different question from the Matrix:

- **Matrix:** What kind of work is this?
- **Queue:** What should I execute first?

Queue modes:

- **Manual** — drag and reorder tasks yourself
- **Importance** — sort primarily by importance
- **Urgency** — sort primarily by effective urgency

The **oldest active AP is always pinned to Priority #1**, preventing long-neglected work from being silently buried.

### Daily Check

Daily Check shows enabled recurring checkpoints and whether each one has been completed today. Completion history is date-based, so there is no midnight job that has to rewrite every task.

### Calendar

The Calendar view displays AP deadlines in a monthly layout. Up to several tasks are shown directly in each day cell, with overflow indicated when more exist.

### Tags

Tags provide a fast context filter for active APs. The Tags view shows tag usage counts and lets you narrow the AP list without changing the underlying task state.

### Statistics

Statistics provide lightweight personal signals rather than productivity scoring. Current metrics include:

- active AP count
- completed/archive count
- today's Daily Check progress
- Jira Watch count
- workload distribution across Q1-Q4

### Search and editing

APs can be searched and edited from the dashboard. Editors are shown in drawers so the main Matrix stays visible and the app remains focused on the current workload.

### Local-first storage

Planner data is stored locally as JSON.

The backend provides:

- atomic file replacement
- `.bak` recovery backup
- cross-platform sidecar file locking
- configurable data storage location
- automatic legacy schema migration

The goal is to keep the app usable without a cloud account or hosted database.

---

# Downloads and package types

EISEN / DAY intentionally separates **source code** from **end-user releases**.

| Package | Intended for | Contents |
| --- | --- | --- |
| `EisenDay-source.zip` | Development / GitHub | `backend/`, `frontend/`, build scripts, tests, README |
| `EisenDay-windows-portable.zip` | Windows users | `EisenDay.exe` only |
| `EisenDay-macos-arm64.zip` / `x86_64.zip` | macOS users | `EisenDay.app` only |

If you only want to use the app, download the package for your operating system. You do **not** need the `backend/` or `frontend/` source folders. Those folders exist only in the developer/source package.

---


# Build on GitHub — Windows or macOS

The repository includes a GitHub Actions workflow at:

```text
.github/workflows/build-desktop.yml
```

You can build **two end-user packages directly on GitHub** without setting up both operating systems yourself:

| GitHub build option | Runner | Output |
| --- | --- | --- |
| **Windows x64** | `windows-latest` | `EisenDay-windows-portable.zip` → `EisenDay.exe` |
| **macOS Apple Silicon** | `macos-15` | `EisenDay-macos-arm64.zip` → `EisenDay.app` |

## Manual build from the GitHub UI

1. Open the repository on GitHub.
2. Go to **Actions**.
3. Select **Build Desktop Releases**.
4. Click **Run workflow**.
5. Choose `all`, `windows`, or `macos`.
6. When the run finishes, download the requested package from the **Artifacts** section.

Selecting `all` builds the Windows and macOS packages in parallel. Selecting one platform builds only that package.

## Automatic GitHub Release

Pushing a tag beginning with `v` builds **both** packages and attaches them to the matching GitHub Release:

```bash
git tag v2.0.0
git push origin v2.0.0
```

The release will contain only the two end-user downloads:

```text
EisenDay-windows-portable.zip
EisenDay-macos-arm64.zip
```

The source code remains in the GitHub repository itself; `backend/` and `frontend/` are not placed inside either release ZIP.

> The macOS GitHub build currently targets **Apple Silicon (arm64)**. Intel Mac can be added later as a third build if needed.

---

# Quick Start

## Option A — Run a packaged app

Use this if you already have a built release.

### Windows

1. Extract `EisenDay-windows-portable.zip`.
2. Double-click `EisenDay.exe`.
3. Start using the app.

You do **not** need Python, Node.js, npm, or a separate backend process.

### macOS

1. Extract `EisenDay-macos-*.zip`.
2. Double-click `EisenDay.app`.
3. If macOS blocks the unsigned local build on first launch, use **Finder → right-click `EisenDay.app` → Open**.

You do **not** need Python, Node.js, npm, or a separate backend process.

The packaged app starts a local FastAPI server on an available `127.0.0.1` port and opens the React UI inside a native desktop window.

---

## Option B — Run from source

### Requirements

- Python 3
- Node.js + npm

Clone or extract the repository, then run commands from the project root.

### macOS — easiest development start

On first use:

```bash
chmod +x run_macos_dev.sh
```

Then run:

```bash
./run_macos_dev.sh
```

The script automatically:

1. creates a Python virtual environment
2. installs Python dependencies
3. installs frontend dependencies
4. builds the React frontend
5. starts the FastAPI backend
6. opens EISEN / DAY in a native window

Use the same command on later runs:

```bash
./run_macos_dev.sh
```

### Windows — development mode

Create the Python environment:

```powershell
py -3 -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install --upgrade pip
pip install -r requirements-dev.txt
```

Start the backend in **Terminal 1**:

```powershell
$env:PYTHONPATH = (Get-Location).Path
uvicorn backend.app.main:create_app --factory --host 127.0.0.1 --port 8787
```

Start the frontend in **Terminal 2**:

```powershell
cd frontend
npm install
$env:VITE_API_MODE='http'
$env:VITE_API_BASE_URL='http://127.0.0.1:8787'
npm run dev
```

Open the localhost URL printed by Vite.

> Development mode runs frontend and backend separately. Packaged releases do not.

---

# Build a desktop release

## macOS `.app`

Run on macOS:

```bash
chmod +x build_macos.sh
./build_macos.sh
```

Outputs:

```text
dist/EisenDay.app
release/EisenDay-macos-arm64.zip   # Apple Silicon
release/EisenDay-macos-x86_64.zip  # Intel Mac
```

The script automatically builds for the current Mac architecture.

Optional universal build:

```bash
EISEN_DAY_TARGET_ARCH=universal2 ./build_macos.sh
```

`universal2` requires a universal2 Python installation and compatible native dependencies.

## Windows `.exe`

Run on Windows PowerShell:

```powershell
.\build_windows.ps1
```

Outputs:

```text
dist\EisenDay.exe
release\EisenDay-windows-portable.zip
```

The build script performs the frontend build, backend tests, PyInstaller packaging, and portable ZIP creation.

---

# Architecture

```text
EisenDay.exe / EisenDay.app
        │
        ├── pywebview native desktop window
        │
        └── FastAPI on 127.0.0.1:{available_port}
                │
                ├── React/Vite production UI
                ├── REST API
                ├── AP / Matrix logic
                ├── Queue logic
                ├── Daily Checkpoint logic
                ├── Jira REST client
                └── local JSON storage
```

Production uses same-origin `/api/...` requests, so the launcher can choose a free localhost port dynamically without hard-coding a frontend API address.

---

# Project structure

```text
eisen_day/
├── frontend/               React + TypeScript + Vite
│   ├── src/
│   │   ├── api/            frontend API adapter
│   │   ├── components/     dashboard, matrix, drawers, views
│   │   ├── state/          planner state/provider
│   │   ├── types/          TypeScript domain models
│   │   └── utils/          urgency, quadrant, queue helpers
│   └── API_CONTRACT.md
│
├── backend/                FastAPI backend
│   ├── app/                API, services, storage, Jira logic
│   └── tests/
│
├── launcher.py             desktop entry point
├── build_windows.ps1       Windows release build
├── build_macos.sh          macOS release build
├── run_macos_dev.sh        macOS source-run helper
├── EisenDay.spec           Windows PyInstaller spec
├── EisenDay_macos.spec     macOS PyInstaller spec
├── requirements.txt
├── requirements-dev.txt
└── requirements-macos.txt
```

---

# Local data locations

## Windows

```text
%APPDATA%\EisenDay\
  config.json
  jira-settings.json
  data\eisenhower.json
```

## macOS

```text
~/Library/Application Support/EisenDay/
  config.json
  jira-settings.json
  data/eisenhower.json
```

`eisenhower.json` is written using atomic replacement. When a valid previous file exists, a `.bak` backup is kept for recovery.

A sidecar `.lock` file prevents multiple processes from writing the same planner file at the same time.

The data location can also be changed from **Settings**.

---

# Jira setup

Open **Settings → Jira** inside EISEN / DAY.

Supported configuration:

- Jira Base URL
- Bearer authentication
- Basic authentication
- Jira REST API v2 or v3
- API token

The Jira token is stored separately in `jira-settings.json` and is **never returned by `/api/state`**.

During sync:

- AP-linked Jira metadata is refreshed
- Jira Watch items are updated
- issues in Jira's `Done` status category are removed
- completed local APs can remain tracked when Jira is still open

Jira features require network access to the configured Jira server. The rest of the app can operate locally.

---

# Legacy data migration

EISEN / DAY can migrate data from the previous Go-based version when compatible legacy files are detected.

Current migration support includes:

- `dailyTasks` → `dailyCheckpoints`
- `jiraTickets` → `jiraWatch`
- existing AP and Jira metadata preservation where possible
- legacy `%APPDATA%/EisenhowerTodo` settings discovery on Windows

Before the first schema migration, the original JSON is additionally preserved as:

```text
.pre-v6.bak
```

---

# Backend API overview

Production frontend uses same-origin `/api/...` endpoints.

### Planner

```text
GET    /api/state
POST   /api/ap
PUT    /api/ap/{id}
DELETE /api/ap/{id}
POST   /api/ap/{id}/complete
POST   /api/archive/{id}/restore
POST   /api/queue
```

### Daily Checkpoints

```text
POST   /api/daily-checkpoints
PUT    /api/daily-checkpoints/{id}
DELETE /api/daily-checkpoints/{id}
POST   /api/daily-checkpoints/{id}/toggle
```

### Jira

```text
PUT    /api/jira/watch
POST   /api/jira/watch/sync
GET    /api/jira/config
PUT    /api/jira/config
POST   /api/jira/test
```

### Settings

```text
PUT    /api/settings
GET    /api/info
```

See [`frontend/API_CONTRACT.md`](frontend/API_CONTRACT.md) for the frontend-facing contract.

---

# Tests

Run backend tests from the project root.

### macOS / Linux

```bash
PYTHONPATH=. pytest -q backend/tests
```

### Windows PowerShell

```powershell
$env:PYTHONPATH = (Get-Location).Path
pytest -q backend\tests
```

The current backend test suite covers the core local planner behavior, including AP operations, Daily Checkpoints, queue rules, storage, migration, and Jira-related logic.

---

# Frontend build only

```bash
cd frontend
npm install
npm run build
```

The production frontend communicates with the packaged FastAPI server through same-origin `/api/...` routes.

---

# Troubleshooting

### macOS: `permission denied: ./run_macos_dev.sh`

```bash
chmod +x run_macos_dev.sh build_macos.sh
```

### macOS blocks `EisenDay.app`

For an unsigned local build, use:

**Finder → right-click `EisenDay.app` → Open**

There is no need to disable Gatekeeper system-wide.

### Windows PowerShell blocks virtual-environment activation

You can call the virtual environment's Python directly:

```powershell
.\.venv\Scripts\python.exe -m pip install -r requirements-dev.txt
```

### Port `8787` is already in use during development

Choose another backend port and update `VITE_API_BASE_URL` to match.

Packaged `EisenDay.exe` and `EisenDay.app` automatically choose an available localhost port.

### The app is already running

EISEN / DAY uses a local lock to protect the planner data file from concurrent writes. If another instance is already using the same data file, a second process may be blocked to avoid corruption.

---

# Notes

- Core planner functionality is designed to work locally without an internet connection.
- Jira integration requires access to the configured Jira server.
- Windows executables must be built on Windows.
- macOS `.app` bundles must be built on macOS.
- macOS Developer ID signing and Apple notarization require an Apple Developer distribution environment and are not performed automatically by the local build script.

---

## Design principle

**The Matrix classifies work. The Queue decides what to do next. Daily Checkpoints protect recurring operational habits. Jira Watch keeps external work from disappearing.**

That separation is the core idea behind EISEN / DAY.
