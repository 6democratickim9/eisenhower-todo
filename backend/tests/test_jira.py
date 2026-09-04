from __future__ import annotations

import json

import httpx

from backend.app.jira import JiraClient, browse_url, flatten_comment, map_watch_status, normalize_base_url
from backend.app.models import JiraSecretConfig


def test_jira_parse_and_adf_comment():
    payload = {
        "fields": {
            "summary": "Deployment issue",
            "status": {"name": "In Progress", "statusCategory": {"key": "indeterminate"}},
            "assignee": {"displayName": "Minju"},
            "created": "2026-09-01T00:00:00.000+0000",
            "updated": "2026-09-03T00:00:00.000+0000",
            "comment": {
                "comments": [
                    {"body": {"type": "doc", "content": [{"type": "paragraph", "content": [{"type": "text", "text": "staging test done"}]}]}}
                ]
            },
        }
    }

    def handler(request: httpx.Request):
        assert request.headers["Authorization"] == "Bearer secret"
        return httpx.Response(200, json=payload, headers={"content-type": "application/json"})

    client = JiraClient(transport=httpx.MockTransport(handler))
    try:
        config = JiraSecretConfig(baseUrl="https://jira.example.com", token="secret")
        issue = client.fetch_issue(config, "ops-123")
        assert issue.issue_key == "OPS-123"
        assert issue.assignee == "Minju"
        assert issue.last_comment == "staging test done"
        assert map_watch_status(issue.raw_status, issue.status_category) == "In Review"
    finally:
        client.close()


def test_flatten_comment_plain_and_nested():
    assert flatten_comment(" hello ") == "hello"
    assert "hello" in flatten_comment({"type": "doc", "content": [{"type": "paragraph", "content": [{"text": "hello"}]}]})


def test_jira_base_url_normalizes_ticket_and_rest_urls():
    assert normalize_base_url("https://jira.example.com/browse/OPS-123?focusedCommentId=1") == "https://jira.example.com"
    assert normalize_base_url("https://jira.example.com/jira/rest/api/2/issue/OPS-123") == "https://jira.example.com/jira"
    assert normalize_base_url("jira.example.com/jira/") == "https://jira.example.com/jira"
    assert browse_url("https://jira.example.com/jira/browse/OPS-123", "OPS-456") == "https://jira.example.com/jira/browse/OPS-456"
