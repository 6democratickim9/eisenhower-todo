from __future__ import annotations

import base64
import re
from typing import Any
from urllib.parse import quote, urlparse, urlunparse

import httpx

from .models import JiraIssueData, JiraSecretConfig, JiraWatchStatus


class JiraError(RuntimeError):
    pass


def normalize_base_url(raw: str) -> str:
    value = raw.strip().rstrip("/")
    if not value:
        return ""
    parsed = urlparse(value if "://" in value else "https://" + value)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise ValueError("Jira Base URL이 올바르지 않습니다.")

    # Users often paste a ticket URL or REST endpoint instead of the Jira root.
    # Preserve an optional Jira context path (for example /jira), but strip
    # ticket/API suffixes so Issue Key links always become <base>/browse/KEY.
    path = parsed.path.rstrip("/")
    lower = path.casefold()
    for marker in ("/browse/", "/rest/api/"):
        index = lower.find(marker)
        if index >= 0:
            path = path[:index]
            lower = path.casefold()
            break
    if lower.endswith("/browse"):
        path = path[: -len("/browse")]

    return urlunparse((parsed.scheme, parsed.netloc, path.rstrip("/"), "", "", "")).rstrip("/")


def extract_issue_key(value: str) -> str:
    raw = value.strip().upper()
    match = re.search(r"(?:^|/)([A-Z][A-Z0-9_]+-\d+)(?:$|[/?#])", raw)
    return match.group(1) if match else raw if re.fullmatch(r"[A-Z][A-Z0-9_]+-\d+", raw) else ""


def browse_url(base_url: str, issue_key: str) -> str:
    return f"{normalize_base_url(base_url)}/browse/{quote(issue_key)}" if base_url and issue_key else ""


def map_watch_status(raw_status: str, status_category: str) -> JiraWatchStatus:
    category = status_category.strip().casefold()
    raw = raw_status.strip().casefold()
    if category in {"done", "complete", "completed"} or raw in {"done", "closed", "resolved", "complete", "completed"}:
        return "Done"
    if "block" in raw or "hold" in raw or "imped" in raw:
        return "Blocked"
    if category in {"indeterminate", "in progress", "in_progress"} or any(token in raw for token in ["progress", "review", "testing", "qa", "develop"]):
        return "In Review"
    return "Open"


def _display_name(value: Any) -> str:
    if not isinstance(value, dict):
        return ""
    for key in ("displayName", "name", "emailAddress", "accountId"):
        if value.get(key):
            return str(value[key])
    return ""


def flatten_comment(value: Any) -> str:
    if isinstance(value, str):
        return value.strip()
    if isinstance(value, list):
        return "".join(flatten_comment(x) for x in value).strip()
    if not isinstance(value, dict):
        return ""
    if value.get("type") == "hardBreak":
        return "\n"
    if isinstance(value.get("text"), str):
        return value["text"]
    joined = "".join(flatten_comment(x) for x in value.get("content", []) if x is not None)
    if value.get("type") in {"paragraph", "heading", "listItem", "blockquote", "bulletList", "orderedList", "doc"}:
        return joined.strip() + "\n"
    return joined


class JiraClient:
    def __init__(self, timeout: float = 12.0, transport: httpx.BaseTransport | None = None):
        self.client = httpx.Client(timeout=timeout, follow_redirects=False, transport=transport)

    def close(self) -> None:
        self.client.close()

    def _headers(self, config: JiraSecretConfig) -> dict[str, str]:
        headers = {"Accept": "application/json"}
        if config.auth_mode == "basic":
            token = base64.b64encode(f"{config.username}:{config.token}".encode()).decode()
            headers["Authorization"] = "Basic " + token
        else:
            headers["Authorization"] = "Bearer " + config.token
        return headers

    def configured(self, config: JiraSecretConfig) -> bool:
        return bool(config.base_url.strip() and config.token.strip())

    def test(self, config: JiraSecretConfig) -> str:
        if not self.configured(config):
            raise JiraError("Jira Base URL과 token을 먼저 설정하세요.")
        endpoint = normalize_base_url(config.base_url) + f"/rest/api/{config.api_version}/myself"
        response = self.client.get(endpoint, headers=self._headers(config))
        self._raise_problem(response)
        payload = response.json()
        name = _display_name(payload)
        return f"Jira 연결 성공{': ' + name if name else ''}"

    def fetch_issue(self, config: JiraSecretConfig, issue_key: str) -> JiraIssueData:
        key = extract_issue_key(issue_key)
        if not key:
            raise JiraError("Jira issue key를 찾을 수 없습니다.")
        if not self.configured(config):
            raise JiraError("Jira Base URL과 token을 먼저 설정하세요.")
        base = normalize_base_url(config.base_url)
        endpoint = base + f"/rest/api/{config.api_version}/issue/{quote(key)}"
        fields = "summary,status,assignee,comment,created,updated"
        response = self.client.get(endpoint, params={"fields": fields}, headers=self._headers(config))
        self._raise_problem(response)
        try:
            payload = response.json()
        except ValueError as exc:
            raise JiraError("Jira가 JSON이 아닌 응답을 반환했습니다.") from exc
        fields_obj = payload.get("fields") if isinstance(payload, dict) else {}
        fields_obj = fields_obj if isinstance(fields_obj, dict) else {}
        status_obj = fields_obj.get("status") if isinstance(fields_obj.get("status"), dict) else {}
        category_obj = status_obj.get("statusCategory") if isinstance(status_obj.get("statusCategory"), dict) else {}
        comments_obj = fields_obj.get("comment") if isinstance(fields_obj.get("comment"), dict) else {}
        comments = comments_obj.get("comments") if isinstance(comments_obj.get("comments"), list) else []
        last_comment = ""
        if comments:
            last = comments[-1] if isinstance(comments[-1], dict) else {}
            last_comment = flatten_comment(last.get("body")).strip()
        return JiraIssueData(
            issueKey=key,
            title=str(fields_obj.get("summary") or ""),
            rawStatus=str(status_obj.get("name") or ""),
            statusCategory=str(category_obj.get("key") or category_obj.get("name") or ""),
            assignee=_display_name(fields_obj.get("assignee")),
            lastComment=last_comment,
            createdAt=str(fields_obj.get("created") or ""),
            updatedAt=str(fields_obj.get("updated") or ""),
            url=browse_url(base, key),
        )

    @staticmethod
    def _raise_problem(response: httpx.Response) -> None:
        status = response.status_code
        content_type = response.headers.get("content-type", "").lower()
        body = response.text.lstrip().lower()
        if 300 <= status < 400:
            target = response.headers.get("location", "")
            raise JiraError(f"Jira 인증 리다이렉트 (HTTP {status}{' → ' + target if target else ''}). Base URL/인증 방식을 확인하세요.")
        if status == 401:
            raise JiraError("Jira 인증 실패 (HTTP 401). Token 또는 인증 방식을 확인하세요.")
        if status == 403:
            raise JiraError("Jira 접근 거부 (HTTP 403). Token 권한 또는 이슈 접근 권한을 확인하세요.")
        if status == 404:
            raise JiraError("Jira API/이슈를 찾을 수 없습니다 (HTTP 404). Base URL, Issue Key, API 버전을 확인하세요.")
        if status < 200 or status >= 300:
            raise JiraError(f"Jira HTTP {status}")
        if "text/html" in content_type or body.startswith("<!doctype") or body.startswith("<html"):
            raise JiraError("Jira가 JSON 대신 HTML을 반환했습니다. SSO/login redirect 가능성이 있습니다.")
