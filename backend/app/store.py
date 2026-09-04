from __future__ import annotations

import copy
import json
import os
import threading
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from pydantic import ValidationError

from .jira import JiraClient, JiraError, browse_url, extract_issue_key, map_watch_status, normalize_base_url
from .locking import DataFileInUseError, FileLock
from .models import (
    APDraft,
    ActionPoint,
    DailyCheckpoint,
    DailyCheckpointDraft,
    JiraConfigPublic,
    JiraConfigUpdate,
    JiraIssueData,
    JiraLink,
    JiraSecretConfig,
    JiraWatchItem,
    PlannerData,
    PlannerSettings,
    PlannerState,
    QueueMode,
)
from .paths import ResolvedPaths, normalize_data_path
from .persistence import atomic_write_json, load_json_with_recovery


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def default_display_name() -> str:
    return (os.environ.get("USERNAME") or os.environ.get("USER") or "You").strip() or "You"


def _parse_iso(value: str | None) -> datetime:
    if not value:
        return datetime.max.replace(tzinfo=timezone.utc)
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return datetime.max.replace(tzinfo=timezone.utc)


def _deadline_pressure(deadline: str | None) -> float:
    if not deadline:
        return 0.0
    due = _parse_iso(deadline)
    if due.year >= 9999:
        return 0.0
    now = datetime.now(timezone.utc)
    if due.tzinfo is None:
        due = due.replace(tzinfo=timezone.utc)
    hours = (due.astimezone(timezone.utc) - now).total_seconds() / 3600
    if hours <= 0:
        return 80.0
    if hours >= 168:
        return 0.0
    if hours >= 72:
        return 15 * ((168 - hours) / 96)
    if hours >= 24:
        return 15 + (35 - 15) * ((72 - hours) / 48)
    return 35 + (70 - 35) * ((24 - hours) / 24)


def _effective_urgency(ap: ActionPoint) -> int:
    return round(max(0, min(100, ap.base_urgency + _deadline_pressure(ap.deadline))))


def _dump(model: Any) -> dict[str, Any]:
    return model.model_dump(by_alias=True, exclude_none=True)


class PlannerStore:
    def __init__(self, paths: ResolvedPaths, jira_client: JiraClient | None = None):
        self.paths = paths
        self._mutex = threading.RLock()
        self._lock = FileLock(paths.data_path)
        self._lock.acquire()
        self.jira_client = jira_client or JiraClient()
        try:
            self.settings = self._load_settings()
            self.jira_config = self._load_jira_config()
            if not self.jira_config.base_url and self.settings.jira_base_url:
                self.jira_config.base_url = self.settings.jira_base_url
            self.data = self._load_data()
        except Exception:
            self._lock.release()
            self.jira_client.close()
            raise

    def close(self) -> None:
        self.jira_client.close()
        self._lock.release()

    def _load_settings(self) -> PlannerSettings:
        raw: dict[str, Any] = {}
        if self.paths.config_path.exists():
            try:
                loaded = json.loads(self.paths.config_path.read_text(encoding="utf-8"))
                if isinstance(loaded, dict):
                    raw = loaded
            except (OSError, json.JSONDecodeError):
                raw = {}
        return PlannerSettings(
            displayName=str(raw.get("displayName") or default_display_name()),
            timezone=str(raw.get("timezone") or "local"),
            jiraBaseUrl=str(raw.get("jiraBaseUrl") or ""),
            dataPath=str(self.paths.data_path),
            autoSyncMinutes=int(raw.get("autoSyncMinutes") or 15),
        )

    def _save_settings_locked(self) -> None:
        value = _dump(self.settings)
        value["dataPath"] = str(self.paths.data_path)
        atomic_write_json(self.paths.config_path, value, backup=True)

    def _load_jira_config(self) -> JiraSecretConfig:
        env_token = os.environ.get("EISEN_JIRA_TOKEN", "")
        env_base = os.environ.get("EISEN_JIRA_BASE_URL", "")
        env_username = os.environ.get("EISEN_JIRA_USERNAME", "")
        env_auth = os.environ.get("EISEN_JIRA_AUTH_MODE", "")
        raw: dict[str, Any] = {}
        if self.paths.jira_path.exists():
            try:
                loaded = json.loads(self.paths.jira_path.read_text(encoding="utf-8"))
                if isinstance(loaded, dict):
                    raw = loaded
            except (OSError, json.JSONDecodeError):
                raw = {}
        # old Go key names are intentionally accepted.
        base = env_base or str(raw.get("baseUrl") or raw.get("base_url") or "")
        token = env_token or str(raw.get("token") or "")
        username = env_username or str(raw.get("username") or "")
        auth = (env_auth or str(raw.get("authMode") or "bearer")).lower()
        api_version = str(raw.get("apiVersion") or "2")
        return JiraSecretConfig(
            baseUrl=normalize_base_url(base) if base else "",
            authMode="basic" if auth == "basic" else "bearer",
            username=username.strip(),
            token=token,
            apiVersion="3" if api_version == "3" else "2",
        )

    def _save_jira_config_locked(self) -> None:
        atomic_write_json(self.paths.jira_path, _dump(self.jira_config), backup=True)

    def _empty_data(self) -> PlannerData:
        return PlannerData(version=6, actionPoints=[], archive=[], dailyCheckpoints=[], jiraWatch=[], queueMode="manual")

    def _load_data(self) -> PlannerData:
        raw, source = load_json_with_recovery(self.paths.data_path)
        if raw is None:
            data = self._empty_data()
            atomic_write_json(self.paths.data_path, _dump(data), backup=False)
            return data
        data, migrated = self._migrate_raw(raw)
        if migrated or source != self.paths.data_path:
            if source and source.exists() and source == self.paths.data_path:
                migration_backup = self.paths.data_path.with_suffix(self.paths.data_path.suffix + ".pre-v6.bak")
                if not migration_backup.exists():
                    try:
                        migration_backup.write_bytes(self.paths.data_path.read_bytes())
                    except OSError:
                        pass
            atomic_write_json(self.paths.data_path, _dump(data), backup=True)
        return data

    def _migrate_raw(self, raw: dict[str, Any]) -> tuple[PlannerData, bool]:
        # Current schema.
        if "dailyCheckpoints" in raw or "jiraWatch" in raw:
            current = dict(raw)
            current.setdefault("version", 6)
            current.setdefault("actionPoints", [])
            current.setdefault("archive", [])
            current.setdefault("dailyCheckpoints", [])
            current.setdefault("jiraWatch", [])
            current.setdefault("queueMode", "manual")
            try:
                data = PlannerData.model_validate(current)
            except ValidationError as exc:
                raise ValueError(f"Invalid planner data: {exc}") from exc
            migrated = data.version != 6
            data.version = 6
            return data, migrated

        # v1.x Go schema migration.
        migrated: dict[str, Any] = {
            "version": 6,
            "actionPoints": raw.get("actionPoints") or [],
            "archive": raw.get("archive") or [],
            "dailyCheckpoints": [],
            "jiraWatch": [],
            "queueMode": raw.get("queueMode") or "manual",
        }
        for index, item in enumerate(raw.get("dailyTasks") or []):
            if not isinstance(item, dict):
                continue
            last = str(item.get("lastCheckedDate") or "").strip()
            migrated["dailyCheckpoints"].append(
                {
                    "id": str(item.get("id") or f"dc-{uuid.uuid4()}"),
                    "title": str(item.get("title") or "Daily Checkpoint"),
                    "enabled": not bool(item.get("disabled", False)),
                    "order": index,
                    "completedDates": [last] if last else [],
                    "createdAt": str(item.get("createdAt") or now_iso()),
                    "updatedAt": str(item.get("updatedAt") or item.get("createdAt") or now_iso()),
                }
            )
        for index, item in enumerate(raw.get("jiraTickets") or []):
            if not isinstance(item, dict):
                continue
            jira = item.get("jira") if isinstance(item.get("jira"), dict) else {}
            key = extract_issue_key(str(jira.get("issueKey") or jira.get("url") or ""))
            if not key:
                continue
            raw_status = str(jira.get("status") or "Open")
            category = str(jira.get("statusCategory") or "")
            migrated["jiraWatch"].append(
                {
                    "id": f"jw-{uuid.uuid4()}",
                    "issueKey": key,
                    "title": str(jira.get("summary") or item.get("apTitle") or key),
                    "status": map_watch_status(raw_status, category),
                    "rawStatus": raw_status,
                    "statusCategory": category,
                    "assignee": jira.get("assignee"),
                    "lastComment": jira.get("lastComment"),
                    "createdAt": jira.get("jiraCreated") or item.get("trackedAt"),
                    "updatedAt": str(jira.get("jiraUpdated") or item.get("trackedAt") or now_iso()),
                    "lastSyncedAt": jira.get("lastSyncedAt"),
                    "url": jira.get("url"),
                    "syncError": jira.get("syncError"),
                }
            )
        try:
            return PlannerData.model_validate(migrated), True
        except ValidationError as exc:
            raise ValueError(f"Legacy planner data migration failed: {exc}") from exc

    def _save_data_locked(self) -> None:
        self.data.version = 6
        atomic_write_json(self.paths.data_path, _dump(self.data), backup=True)

    def state(self) -> PlannerState:
        with self._mutex:
            return PlannerState(
                actionPoints=copy.deepcopy(self.data.action_points),
                archive=copy.deepcopy(self.data.archive),
                dailyCheckpoints=copy.deepcopy(self.data.daily_checkpoints),
                jiraWatch=copy.deepcopy(self.data.jira_watch),
                queueMode=self.data.queue_mode,
                settings=copy.deepcopy(self.settings),
            )

    def create_ap(self, draft: APDraft) -> ActionPoint:
        with self._mutex:
            now = now_iso()
            jira = self._canonical_jira(draft.jira)
            ap = ActionPoint(
                id=f"ap-{uuid.uuid4()}",
                title=draft.title,
                details=draft.details,
                importance=draft.importance,
                baseUrgency=draft.base_urgency,
                deadline=draft.deadline,
                tags=draft.tags,
                jira=jira,
                queueOrder=max((x.queue_order for x in self.data.action_points), default=-1) + 1,
                createdAt=now,
                updatedAt=now,
            )
            if ap.jira:
                self._sync_link_best_effort(ap)
            self.data.action_points.append(ap)
            self._pin_oldest_locked()
            self._save_data_locked()
            return copy.deepcopy(ap)

    def update_ap(self, ap_id: str, patch: dict[str, Any]) -> ActionPoint:
        with self._mutex:
            index = self._find_ap(ap_id)
            if index < 0:
                raise KeyError("AP not found")
            current = self.data.action_points[index]
            current_dict = _dump(current)
            protected = {"id", "createdAt", "updatedAt", "completedAt"}
            for key, value in patch.items():
                if key not in protected:
                    current_dict[key] = value
            current_dict["id"] = current.id
            current_dict["createdAt"] = current.created_at
            current_dict["updatedAt"] = now_iso()
            current_dict.pop("completedAt", None)
            try:
                updated = ActionPoint.model_validate(current_dict)
            except ValidationError as exc:
                raise ValueError(str(exc)) from exc
            updated.jira = self._canonical_jira(updated.jira)
            if updated.jira:
                self._sync_link_best_effort(updated)
            self.data.action_points[index] = updated
            self._pin_oldest_locked()
            self._save_data_locked()
            return copy.deepcopy(updated)

    def delete_ap(self, ap_id: str) -> None:
        with self._mutex:
            index = self._find_ap(ap_id)
            if index < 0:
                raise KeyError("AP not found")
            self.data.action_points.pop(index)
            self._compact_queue_locked()
            self._save_data_locked()

    def complete_ap(self, ap_id: str) -> None:
        with self._mutex:
            index = self._find_ap(ap_id)
            if index < 0:
                raise KeyError("AP not found")
            ap = self.data.action_points.pop(index)
            ap.completed_at = now_iso()
            ap.updated_at = ap.completed_at
            if ap.jira:
                self._ensure_watch_for_ap_locked(ap)
            self.data.archive.insert(0, ap)
            self._compact_queue_locked()
            self._save_data_locked()

    def restore_ap(self, ap_id: str) -> None:
        with self._mutex:
            index = next((i for i, item in enumerate(self.data.archive) if item.id == ap_id), -1)
            if index < 0:
                raise KeyError("Archived AP not found")
            ap = self.data.archive.pop(index)
            ap.completed_at = None
            ap.updated_at = now_iso()
            ap.queue_order = max((x.queue_order for x in self.data.action_points), default=-1) + 1
            self.data.action_points.append(ap)
            self._pin_oldest_locked()
            self._save_data_locked()

    def reorder_queue(self, ids: list[str], mode: QueueMode | None = None) -> None:
        with self._mutex:
            existing = {ap.id for ap in self.data.action_points}
            seen: set[str] = set()
            clean_ids = [x for x in ids if x in existing and not (x in seen or seen.add(x))]
            remainder = [ap.id for ap in sorted(self.data.action_points, key=lambda x: x.queue_order) if ap.id not in set(clean_ids)]
            requested = clean_ids + remainder
            oldest = self._oldest_ap_locked()
            if oldest:
                requested = [oldest.id] + [ap_id for ap_id in requested if ap_id != oldest.id]
            order = {ap_id: i for i, ap_id in enumerate(requested)}
            for ap in self.data.action_points:
                ap.queue_order = order[ap.id]
            if mode:
                self.data.queue_mode = mode
            self._save_data_locked()

    def create_daily(self, draft: DailyCheckpointDraft) -> DailyCheckpoint:
        with self._mutex:
            now = now_iso()
            item = DailyCheckpoint(
                id=f"dc-{uuid.uuid4()}",
                title=draft.title,
                enabled=draft.enabled,
                order=max((x.order for x in self.data.daily_checkpoints), default=-1) + 1,
                completedDates=[],
                createdAt=now,
                updatedAt=now,
            )
            self.data.daily_checkpoints.append(item)
            self._save_data_locked()
            return copy.deepcopy(item)

    def update_daily(self, item_id: str, patch: dict[str, Any]) -> DailyCheckpoint:
        with self._mutex:
            index = next((i for i, item in enumerate(self.data.daily_checkpoints) if item.id == item_id), -1)
            if index < 0:
                raise KeyError("Daily checkpoint not found")
            current = self.data.daily_checkpoints[index]
            payload = _dump(current)
            for key, value in patch.items():
                if key not in {"id", "createdAt", "updatedAt"}:
                    payload[key] = value
            payload["id"] = current.id
            payload["createdAt"] = current.created_at
            payload["updatedAt"] = now_iso()
            try:
                updated = DailyCheckpoint.model_validate(payload)
            except ValidationError as exc:
                raise ValueError(str(exc)) from exc
            self.data.daily_checkpoints[index] = updated
            self._save_data_locked()
            return copy.deepcopy(updated)

    def delete_daily(self, item_id: str) -> None:
        with self._mutex:
            before = len(self.data.daily_checkpoints)
            self.data.daily_checkpoints = [x for x in self.data.daily_checkpoints if x.id != item_id]
            if len(self.data.daily_checkpoints) == before:
                raise KeyError("Daily checkpoint not found")
            for index, item in enumerate(sorted(self.data.daily_checkpoints, key=lambda x: x.order)):
                item.order = index
            self._save_data_locked()

    def toggle_daily(self, item_id: str, date_key: str) -> DailyCheckpoint:
        with self._mutex:
            item = next((x for x in self.data.daily_checkpoints if x.id == item_id), None)
            if item is None:
                raise KeyError("Daily checkpoint not found")
            if not item.enabled:
                raise ValueError("Disabled Daily Checkpoint는 체크할 수 없습니다.")
            if date_key in item.completed_dates:
                item.completed_dates = [x for x in item.completed_dates if x != date_key]
            else:
                item.completed_dates = [*item.completed_dates, date_key]
            # Keep history bounded while still allowing statistics over recent months.
            if len(item.completed_dates) > 400:
                item.completed_dates = sorted(item.completed_dates)[-400:]
            item.updated_at = now_iso()
            self._save_data_locked()
            return copy.deepcopy(item)

    def save_jira_watch(self, items: list[JiraWatchItem]) -> None:
        with self._mutex:
            seen: set[str] = set()
            cleaned: list[JiraWatchItem] = []
            for item in items:
                key = extract_issue_key(item.issue_key)
                if not key or key in seen or item.status == "Done":
                    continue
                seen.add(key)
                item.issue_key = key
                if self.jira_config.base_url:
                    item.url = browse_url(self.jira_config.base_url, key)
                cleaned.append(item)
            self.data.jira_watch = cleaned
            self._save_data_locked()

    def sync_jira_watch(self) -> list[JiraWatchItem]:
        with self._mutex:
            changed = False
            output: list[JiraWatchItem] = []
            for item in self.data.jira_watch:
                synced = self._sync_watch_item_best_effort(item)
                changed = True
                if synced.status != "Done":
                    output.append(synced)
            self.data.jira_watch = output
            # Also refresh Jira metadata attached to active APs, but do not auto-add removed watch items.
            for ap in self.data.action_points:
                if ap.jira:
                    self._sync_link_best_effort(ap)
                    changed = True
            if changed:
                self._save_data_locked()
            return copy.deepcopy(self.data.jira_watch)

    def save_settings(self, new_settings: PlannerSettings) -> PlannerSettings:
        with self._mutex:
            target = normalize_data_path(new_settings.data_path) if new_settings.data_path.strip() else self.paths.data_path
            if target != self.paths.data_path:
                self._switch_data_path_locked(target)
            self.settings = PlannerSettings(
                displayName=new_settings.display_name.strip() or default_display_name(),
                timezone=new_settings.timezone.strip() or "local",
                jiraBaseUrl=normalize_base_url(new_settings.jira_base_url) if new_settings.jira_base_url.strip() else "",
                dataPath=str(self.paths.data_path),
                autoSyncMinutes=new_settings.auto_sync_minutes,
            )
            if self.settings.jira_base_url != self.jira_config.base_url:
                self.jira_config.base_url = self.settings.jira_base_url
                self._save_jira_config_locked()
            self._save_settings_locked()
            return copy.deepcopy(self.settings)

    def jira_public_config(self) -> JiraConfigPublic:
        with self._mutex:
            return JiraConfigPublic(
                baseUrl=self.jira_config.base_url,
                authMode=self.jira_config.auth_mode,
                username=self.jira_config.username,
                apiVersion=self.jira_config.api_version,
                tokenConfigured=bool(self.jira_config.token),
            )

    def save_jira_config(self, patch: JiraConfigUpdate) -> JiraConfigPublic:
        with self._mutex:
            payload = patch.model_dump(exclude_unset=True)
            if "base_url" in payload and payload["base_url"] is not None:
                self.jira_config.base_url = normalize_base_url(payload["base_url"]) if str(payload["base_url"]).strip() else ""
            if payload.get("auth_mode"):
                self.jira_config.auth_mode = payload["auth_mode"]
            if "username" in payload and payload["username"] is not None:
                self.jira_config.username = str(payload["username"]).strip()
            if payload.get("api_version"):
                self.jira_config.api_version = payload["api_version"]
            if "token" in payload and payload["token"] is not None and str(payload["token"]).strip():
                self.jira_config.token = str(payload["token"]).strip()
            self.settings.jira_base_url = self.jira_config.base_url
            self._save_jira_config_locked()
            self._save_settings_locked()
            return self.jira_public_config()

    def test_jira(self) -> str:
        with self._mutex:
            return self.jira_client.test(copy.deepcopy(self.jira_config))

    def _switch_data_path_locked(self, target: Path) -> None:
        target.parent.mkdir(parents=True, exist_ok=True)
        new_lock = FileLock(target)
        new_lock.acquire()
        try:
            if target.exists():
                raw, _ = load_json_with_recovery(target)
                if raw is None:
                    raise ValueError("선택한 JSON 파일을 읽을 수 없습니다.")
                new_data, migrated = self._migrate_raw(raw)
                if migrated:
                    atomic_write_json(target, _dump(new_data), backup=True)
            else:
                new_data = copy.deepcopy(self.data)
                atomic_write_json(target, _dump(new_data), backup=False)
            old_lock = self._lock
            self._lock = new_lock
            self.paths = ResolvedPaths(self.paths.config_dir, self.paths.config_path, self.paths.jira_path, target, self.paths.running_path)
            self.data = new_data
            self.settings.data_path = str(target)
            self._save_settings_locked()
            old_lock.release()
        except Exception:
            new_lock.release()
            raise

    def _find_ap(self, ap_id: str) -> int:
        return next((i for i, item in enumerate(self.data.action_points) if item.id == ap_id), -1)

    def _oldest_ap_locked(self) -> ActionPoint | None:
        if not self.data.action_points:
            return None
        return min(
            self.data.action_points,
            key=lambda ap: (
                _parse_iso(ap.created_at),
                -ap.importance,
                -_effective_urgency(ap),
                ap.queue_order,
                ap.id,
            ),
        )

    def _pin_oldest_locked(self) -> None:
        oldest = self._oldest_ap_locked()
        if oldest is None:
            return
        rest = [ap for ap in sorted(self.data.action_points, key=lambda x: x.queue_order) if ap.id != oldest.id]
        oldest.queue_order = 0
        for index, ap in enumerate(rest, start=1):
            ap.queue_order = index

    def _compact_queue_locked(self) -> None:
        self._pin_oldest_locked()

    def _canonical_jira(self, jira: JiraLink | None) -> JiraLink | None:
        if jira is None:
            return None
        key = extract_issue_key(jira.issue_key or jira.url or "")
        if not key:
            return None
        jira.issue_key = key
        if self.jira_config.base_url:
            jira.url = browse_url(self.jira_config.base_url, key)
        return jira

    def _apply_issue_to_link(self, link: JiraLink, issue: JiraIssueData) -> None:
        link.issue_key = issue.issue_key
        link.url = issue.url
        link.summary = issue.title
        link.status = issue.raw_status
        link.status_category = issue.status_category
        link.assignee = issue.assignee
        link.last_comment = issue.last_comment
        link.jira_created = issue.created_at
        link.jira_updated = issue.updated_at
        link.last_synced_at = now_iso()
        link.sync_error = None

    def _sync_link_best_effort(self, ap: ActionPoint) -> None:
        if not ap.jira:
            return
        ap.jira.last_synced_at = now_iso()
        if not self.jira_client.configured(self.jira_config):
            ap.jira.sync_error = "Jira 인증 설정이 필요합니다."
            return
        try:
            issue = self.jira_client.fetch_issue(self.jira_config, ap.jira.issue_key)
            self._apply_issue_to_link(ap.jira, issue)
        except JiraError as exc:
            ap.jira.sync_error = str(exc)

    def _sync_watch_item_best_effort(self, item: JiraWatchItem) -> JiraWatchItem:
        item.last_synced_at = now_iso()
        if self.jira_config.base_url:
            item.url = browse_url(self.jira_config.base_url, item.issue_key)
        if not self.jira_client.configured(self.jira_config):
            item.sync_error = "Jira 인증 설정이 필요합니다."
            return item
        try:
            issue = self.jira_client.fetch_issue(self.jira_config, item.issue_key)
            item.title = issue.title or item.title
            item.raw_status = issue.raw_status
            item.status_category = issue.status_category
            item.status = map_watch_status(issue.raw_status, issue.status_category)
            item.assignee = issue.assignee
            item.last_comment = issue.last_comment
            item.created_at = issue.created_at or item.created_at
            item.updated_at = issue.updated_at or now_iso()
            item.url = issue.url
            item.sync_error = None
        except JiraError as exc:
            item.sync_error = str(exc)
        return item

    def _ensure_watch_for_ap_locked(self, ap: ActionPoint) -> None:
        if not ap.jira:
            return
        key = extract_issue_key(ap.jira.issue_key)
        if not key:
            return
        category = (ap.jira.status_category or "").casefold()
        if category == "done":
            return
        existing = next((x for x in self.data.jira_watch if x.issue_key.casefold() == key.casefold()), None)
        if existing:
            if ap.jira.summary:
                existing.title = ap.jira.summary
            return
        self.data.jira_watch.append(
            JiraWatchItem(
                id=f"jw-{uuid.uuid4()}",
                issueKey=key,
                title=ap.jira.summary or ap.title,
                status=map_watch_status(ap.jira.status or "", ap.jira.status_category or ""),
                rawStatus=ap.jira.status,
                statusCategory=ap.jira.status_category,
                assignee=ap.jira.assignee,
                lastComment=ap.jira.last_comment,
                createdAt=ap.jira.jira_created or ap.created_at,
                updatedAt=ap.jira.jira_updated or ap.updated_at,
                lastSyncedAt=ap.jira.last_synced_at,
                url=ap.jira.url,
                syncError=ap.jira.sync_error,
            )
        )
