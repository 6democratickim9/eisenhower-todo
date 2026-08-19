# Eisenhower Todo

Local-first Eisenhower Matrix planner for VS Code.

## Key design

- All AP data is stored locally in:
  - `<workspace>/.eisenhower-todo/data.json`
- `.eisenhower-todo/` is gitignored by default.
- Jira link is optional.
- AP details are optional.
- Deadline is optional.
- Unfinished APs remain in the JSON file across days and VS Code restarts.
- Completing an AP moves it to the archive section inside the JSON file.
- Deadline pressure automatically increases urgency, which can move:
  - Q2 -> Q1
  - Q4 -> Q3

## MVP features

- Q1/Q2/Q3/Q4 Eisenhower Matrix
- AP create
- Optional details
- Optional Jira key + URL
- Optional deadline
- Drag & drop between quadrants
- Today's Queue
- Manual queue reorder
- Complete -> archive
- Local JSON persistence
- Auto urgency calculation

## Run locally

```bash
npm install
npm run compile
```

Then open this project in VS Code and press `F5`.

In the Extension Development Host:

1. Open any workspace folder.
2. Run Command Palette.
3. Select `Eisenhower Todo: Open Planner`.

The workspace will get:

```text
.eisenhower-todo/
└── data.json
```

This folder is ignored by Git and should not be committed.

## Data model

Only `title` is conceptually required for an AP.

```ts
interface ActionPoint {
  id: string;
  title: string;

  details?: string;
  deadline?: string;

  importance: number;
  baseUrgency: number;

  jira?: {
    issueKey?: string;
    url: string;
  };

  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  queueOrder?: number;
  history: HistoryEntry[];
}
```

## Jira

The MVP stores only optional Jira metadata.

It intentionally does **not** call Jira APIs yet.

A later version can add an internal-network Jira adapter for:

- ticket status
- assignee
- latest comments
- history timeline

Credentials should use VS Code `SecretStorage`, never the planner JSON file.
