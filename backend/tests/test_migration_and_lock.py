from __future__ import annotations

import json
from pathlib import Path

import pytest

from backend.app.locking import DataFileInUseError
from backend.app.paths import ResolvedPaths
from backend.app.store import PlannerStore


def test_legacy_go_schema_migrates(paths: ResolvedPaths):
    legacy = {
        "version": 5,
        "timezone": "local",
        "queueMode": "manual",
        "actionPoints": [
            {
                "id": "ap-old",
                "title": "Old AP",
                "details": "legacy",
                "importance": 80,
                "baseUrgency": 20,
                "createdAt": "2026-08-01T00:00:00Z",
                "updatedAt": "2026-08-01T00:00:00Z",
                "queueOrder": 0,
                "tags": ["Ops"],
                "jira": {"issueKey": "OPS-1", "url": "https://jira.example/browse/OPS-1", "status": "In Progress"},
                "history": [],
            }
        ],
        "archive": [],
        "dailyTasks": [
            {"id": "daily-1", "title": "Batch check", "createdAt": "2026-08-01T00:00:00Z", "lastCheckedDate": "2026-09-02"}
        ],
        "jiraTickets": [
            {
                "apTitle": "Old AP",
                "trackedAt": "2026-08-01T00:00:00Z",
                "jira": {
                    "issueKey": "OPS-1",
                    "url": "https://jira.example/browse/OPS-1",
                    "summary": "Old AP jira",
                    "status": "In Progress",
                    "statusCategory": "indeterminate",
                },
            }
        ],
    }
    paths.data_path.write_text(json.dumps(legacy), encoding="utf-8")
    store = PlannerStore(paths)
    try:
        state = store.state()
        assert state.action_points[0].id == "ap-old"
        assert state.daily_checkpoints[0].completed_dates == ["2026-09-02"]
        assert state.jira_watch[0].issue_key == "OPS-1"
        assert state.jira_watch[0].status == "In Review"
        saved = json.loads(paths.data_path.read_text(encoding="utf-8"))
        assert saved["version"] == 6
        assert "dailyCheckpoints" in saved and "dailyTasks" not in saved
        assert paths.data_path.with_suffix(".json.pre-v6.bak").exists()
    finally:
        store.close()


def test_sidecar_file_lock_blocks_second_store(paths: ResolvedPaths):
    first = PlannerStore(paths)
    try:
        with pytest.raises(DataFileInUseError):
            PlannerStore(paths)
    finally:
        first.close()
