from __future__ import annotations

from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from backend.app.main import create_app
from backend.app.paths import ResolvedPaths


@pytest.fixture
def paths(tmp_path: Path) -> ResolvedPaths:
    cfg = tmp_path / "config"
    cfg.mkdir()
    data = cfg / "data" / "eisenhower.json"
    data.parent.mkdir()
    return ResolvedPaths(
        config_dir=cfg,
        config_path=cfg / "config.json",
        jira_path=cfg / "jira-settings.json",
        data_path=data,
        running_path=cfg / "running.json",
    )


@pytest.fixture
def client(paths: ResolvedPaths):
    app = create_app(paths, start_auto_sync=False)
    with TestClient(app) as test_client:
        yield test_client
