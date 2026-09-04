from __future__ import annotations

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator


def to_camel(value: str) -> str:
    head, *tail = value.split("_")
    return head + "".join(part[:1].upper() + part[1:] for part in tail)


class ApiModel(BaseModel):
    model_config = ConfigDict(alias_generator=to_camel, populate_by_name=True, extra="ignore")


Quadrant = Literal["Q1", "Q2", "Q3", "Q4"]
QueueMode = Literal["manual", "important", "emergency"]
JiraWatchStatus = Literal["Open", "In Review", "Blocked", "Done"]
JiraAuthMode = Literal["bearer", "basic"]


class JiraLink(ApiModel):
    issue_key: str = ""
    url: str | None = None
    summary: str | None = None
    status: str | None = None
    status_category: str | None = None
    assignee: str | None = None
    last_comment: str | None = None
    jira_created: str | None = None
    jira_updated: str | None = None
    last_synced_at: str | None = None
    sync_error: str | None = None

    @field_validator("issue_key")
    @classmethod
    def normalize_key(cls, value: str) -> str:
        return value.strip().upper()


class ActionPoint(ApiModel):
    id: str
    title: str
    details: str = ""
    importance: int = Field(default=70, ge=0, le=100)
    base_urgency: int = Field(default=40, ge=0, le=100)
    deadline: str | None = None
    tags: list[str] = Field(default_factory=list)
    jira: JiraLink | None = None
    queue_order: int = 0
    created_at: str
    updated_at: str
    completed_at: str | None = None

    @field_validator("title")
    @classmethod
    def title_required(cls, value: str) -> str:
        value = value.strip()
        if not value:
            raise ValueError("AP title is required")
        if len(value) > 200:
            raise ValueError("AP title must be 200 characters or fewer")
        return value

    @field_validator("tags")
    @classmethod
    def normalize_tags(cls, values: list[str]) -> list[str]:
        seen: set[str] = set()
        result: list[str] = []
        for raw in values:
            tag = raw.strip()
            key = tag.casefold()
            if tag and key not in seen:
                result.append(tag)
                seen.add(key)
        return result


class APDraft(ApiModel):
    title: str
    details: str = ""
    importance: int = Field(default=70, ge=0, le=100)
    base_urgency: int = Field(default=40, ge=0, le=100)
    deadline: str | None = None
    tags: list[str] = Field(default_factory=list)
    jira: JiraLink | None = None

    @field_validator("title")
    @classmethod
    def title_required(cls, value: str) -> str:
        value = value.strip()
        if not value:
            raise ValueError("AP title is required")
        if len(value) > 200:
            raise ValueError("AP title must be 200 characters or fewer")
        return value

    @field_validator("tags")
    @classmethod
    def normalize_tags(cls, values: list[str]) -> list[str]:
        seen: set[str] = set()
        result: list[str] = []
        for raw in values:
            tag = raw.strip()
            key = tag.casefold()
            if tag and key not in seen:
                result.append(tag)
                seen.add(key)
        return result


class DailyCheckpoint(ApiModel):
    id: str
    title: str
    enabled: bool = True
    order: int = 0
    completed_dates: list[str] = Field(default_factory=list)
    created_at: str
    updated_at: str

    @field_validator("title")
    @classmethod
    def title_required(cls, value: str) -> str:
        value = value.strip()
        if not value:
            raise ValueError("Daily checkpoint title is required")
        if len(value) > 120:
            raise ValueError("Daily checkpoint title must be 120 characters or fewer")
        return value

    @field_validator("completed_dates")
    @classmethod
    def unique_dates(cls, values: list[str]) -> list[str]:
        return list(dict.fromkeys(v.strip() for v in values if v.strip()))


class DailyCheckpointDraft(ApiModel):
    title: str
    enabled: bool = True

    @field_validator("title")
    @classmethod
    def title_required(cls, value: str) -> str:
        value = value.strip()
        if not value:
            raise ValueError("Daily checkpoint title is required")
        if len(value) > 120:
            raise ValueError("Daily checkpoint title must be 120 characters or fewer")
        return value


class JiraWatchItem(ApiModel):
    id: str
    issue_key: str
    title: str = ""
    status: JiraWatchStatus = "Open"
    raw_status: str | None = None
    status_category: str | None = None
    assignee: str | None = None
    last_comment: str | None = None
    created_at: str | None = None
    updated_at: str
    last_synced_at: str | None = None
    url: str | None = None
    sync_error: str | None = None

    @field_validator("issue_key")
    @classmethod
    def normalize_key(cls, value: str) -> str:
        value = value.strip().upper()
        if not value:
            raise ValueError("Jira issue key is required")
        return value


class PlannerSettings(ApiModel):
    display_name: str = "You"
    timezone: str = "local"
    jira_base_url: str = ""
    data_path: str = ""
    auto_sync_minutes: int = Field(default=15, ge=1, le=1440)


class PlannerData(ApiModel):
    version: int = 6
    action_points: list[ActionPoint] = Field(default_factory=list)
    archive: list[ActionPoint] = Field(default_factory=list)
    daily_checkpoints: list[DailyCheckpoint] = Field(default_factory=list)
    jira_watch: list[JiraWatchItem] = Field(default_factory=list)
    queue_mode: QueueMode = "manual"


class PlannerState(ApiModel):
    action_points: list[ActionPoint] = Field(default_factory=list)
    archive: list[ActionPoint] = Field(default_factory=list)
    daily_checkpoints: list[DailyCheckpoint] = Field(default_factory=list)
    jira_watch: list[JiraWatchItem] = Field(default_factory=list)
    queue_mode: QueueMode = "manual"
    settings: PlannerSettings


class QueueRequest(ApiModel):
    ids: list[str]
    mode: QueueMode | None = None


class DailyToggleRequest(ApiModel):
    date_key: str

    @field_validator("date_key")
    @classmethod
    def validate_date_key(cls, value: str) -> str:
        value = value.strip()
        try:
            datetime.strptime(value, "%Y-%m-%d")
        except ValueError as exc:
            raise ValueError("dateKey must be YYYY-MM-DD") from exc
        return value


class JiraWatchSaveRequest(ApiModel):
    items: list[JiraWatchItem]


class JiraConfigPublic(ApiModel):
    base_url: str = ""
    auth_mode: JiraAuthMode = "bearer"
    username: str = ""
    api_version: Literal["2", "3"] = "2"
    token_configured: bool = False


class JiraConfigUpdate(ApiModel):
    base_url: str | None = None
    auth_mode: JiraAuthMode | None = None
    username: str | None = None
    api_version: Literal["2", "3"] | None = None
    token: str | None = None


class JiraSecretConfig(ApiModel):
    base_url: str = ""
    auth_mode: JiraAuthMode = "bearer"
    username: str = ""
    token: str = ""
    api_version: Literal["2", "3"] = "2"


class JiraIssueData(ApiModel):
    issue_key: str
    title: str = ""
    raw_status: str = ""
    status_category: str = ""
    assignee: str = ""
    last_comment: str = ""
    created_at: str = ""
    updated_at: str = ""
    url: str = ""


class ApiMessage(ApiModel):
    ok: bool = True
    message: str = ""


JsonObject = dict[str, Any]
