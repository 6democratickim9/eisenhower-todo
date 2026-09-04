from __future__ import annotations

import json
import os
import sys
from dataclasses import dataclass
from pathlib import Path

APP_DIR_NAME = "EisenDay"
LEGACY_APP_DIR_NAME = "EisenhowerTodo"


def user_config_home() -> Path:
    if sys.platform == "win32":
        base = os.environ.get("APPDATA") or os.environ.get("LOCALAPPDATA")
        if base:
            return Path(base)
        return Path.home() / "AppData" / "Roaming"
    if sys.platform == "darwin":
        return Path.home() / "Library" / "Application Support"
    return Path(os.environ.get("XDG_CONFIG_HOME", Path.home() / ".config"))


def app_dir() -> Path:
    override = os.environ.get("EISEN_DAY_CONFIG_DIR")
    path = Path(override).expanduser() if override else user_config_home() / APP_DIR_NAME
    path.mkdir(parents=True, exist_ok=True)
    return path


def normalize_data_path(raw: str | Path) -> Path:
    text = os.path.expandvars(os.path.expanduser(str(raw).strip()))
    if not text:
        raise ValueError("JSON 저장 위치를 입력하세요.")
    path = Path(text)
    if not path.is_absolute():
        path = Path.cwd() / path
    if path.exists() and path.is_dir():
        path = path / "eisenhower.json"
    elif not path.suffix:
        path = path / "eisenhower.json"
    if path.suffix.lower() != ".json":
        raise ValueError("저장 파일은 .json 확장자를 사용해야 합니다.")
    return path.resolve()


def _safe_read_json(path: Path) -> dict | None:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
        return value if isinstance(value, dict) else None
    except (OSError, json.JSONDecodeError):
        return None


@dataclass(frozen=True)
class ResolvedPaths:
    config_dir: Path
    config_path: Path
    jira_path: Path
    data_path: Path
    running_path: Path


def resolve_paths() -> ResolvedPaths:
    base = app_dir()
    config_path = base / "config.json"
    jira_path = base / "jira-settings.json"
    running_path = base / "running.json"
    default_data = base / "data" / "eisenhower.json"
    default_data.parent.mkdir(parents=True, exist_ok=True)

    cfg = _safe_read_json(config_path) or {}
    configured = str(cfg.get("dataPath", "")).strip()
    if configured:
        try:
            data_path = normalize_data_path(configured)
            return ResolvedPaths(base, config_path, jira_path, data_path, running_path)
        except ValueError:
            pass

    # One-time compatibility with the previous Go app's stable config folder.
    legacy_base = user_config_home() / LEGACY_APP_DIR_NAME
    legacy_cfg = _safe_read_json(legacy_base / "config.json") or {}
    legacy_data = str(legacy_cfg.get("dataPath", "")).strip()
    if legacy_data:
        try:
            data_path = normalize_data_path(legacy_data)
            config_path.write_text(
                json.dumps(
                    {
                        "dataPath": str(data_path),
                        "displayName": legacy_cfg.get("displayName", "Minju"),
                        "timezone": legacy_cfg.get("timezone", "local"),
                        "autoSyncMinutes": 15,
                    },
                    ensure_ascii=False,
                    indent=2,
                ) + "\n",
                encoding="utf-8",
            )
            old_jira = legacy_base / "jira-settings.json"
            if old_jira.exists() and not jira_path.exists():
                jira_path.write_bytes(old_jira.read_bytes())
            return ResolvedPaths(base, config_path, jira_path, data_path, running_path)
        except (OSError, ValueError):
            pass

    return ResolvedPaths(base, config_path, jira_path, default_data, running_path)
