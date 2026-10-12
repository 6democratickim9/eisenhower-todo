# EISEN / DAY Local API Contract

The React frontend calls `/api/...`. During development Vite proxies `/api` to the local Go backend at `127.0.0.1:8787`.

## Planner

- `GET /api/state` → `PlannerState`
- `POST /api/ap` → create AP
- `PUT /api/ap/{id}` → update AP
- `DELETE /api/ap/{id}`
- `POST /api/ap/{id}/complete`
- `POST /api/archive/{id}/restore`
- `POST /api/queue` body `{ ids, mode? }`

## Daily Checkpoint

- `POST /api/daily-checkpoints`
- `PUT /api/daily-checkpoints/{id}`
- `DELETE /api/daily-checkpoints/{id}`
- `POST /api/daily-checkpoints/{id}/toggle` body `{ dateKey: YYYY-MM-DD }`

## Jira

- `PUT /api/jira/watch` body `{ items: JiraWatchItem[] }`
- `POST /api/jira/watch/sync`
- `GET /api/jira/config` — token itself is never returned
- `PUT /api/jira/config`
- `POST /api/jira/test`

## Settings

- `PUT /api/settings`
- `GET /api/info`

- `PUT /api/note` — save the persistent right-rail quick note (`{ "note": string }`)
