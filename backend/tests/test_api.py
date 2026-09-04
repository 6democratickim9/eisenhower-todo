from __future__ import annotations


def ap_payload(title="Deploy fix"):
    return {
        "title": title,
        "details": "check route",
        "importance": 88,
        "baseUrgency": 70,
        "tags": ["DevOps", "devops", " SRE "],
        "jira": {"issueKey": "ops-123"},
    }


def test_ap_crud_complete_restore_and_queue(client):
    created = client.post("/api/ap", json=ap_payload()).json()
    assert created["title"] == "Deploy fix"
    assert created["tags"] == ["DevOps", "SRE"]
    assert created["jira"]["issueKey"] == "OPS-123"
    assert created["jira"]["syncError"]

    second = client.post("/api/ap", json=ap_payload("Docs")).json()
    assert client.put(f"/api/ap/{created['id']}", json={"importance": 45, "details": "updated"}).json()["importance"] == 45

    assert client.post("/api/queue", json={"ids": [second["id"], created["id"]], "mode": "manual"}).status_code == 204
    state = client.get("/api/state").json()
    order = sorted(state["actionPoints"], key=lambda x: x["queueOrder"])
    assert [x["id"] for x in order] == [created["id"], second["id"]]  # oldest AP is pinned to Priority 1

    assert client.post(f"/api/ap/{created['id']}/complete").status_code == 204
    state = client.get("/api/state").json()
    assert created["id"] not in [x["id"] for x in state["actionPoints"]]
    assert created["id"] in [x["id"] for x in state["archive"]]
    # Jira-linked completed AP is retained in Jira Watch until Jira says Done.
    assert "OPS-123" in [x["issueKey"] for x in state["jiraWatch"]]

    assert client.post(f"/api/archive/{created['id']}/restore").status_code == 204
    state = client.get("/api/state").json()
    assert created["id"] in [x["id"] for x in state["actionPoints"]]

    assert client.delete(f"/api/ap/{second['id']}").status_code == 204


def test_daily_checkpoint_semantics(client):
    item = client.post("/api/daily-checkpoints", json={"title": "Check batch", "enabled": True}).json()
    assert item["completedDates"] == []

    checked = client.post(f"/api/daily-checkpoints/{item['id']}/toggle", json={"dateKey": "2026-09-03"}).json()
    assert checked["completedDates"] == ["2026-09-03"]
    unchecked = client.post(f"/api/daily-checkpoints/{item['id']}/toggle", json={"dateKey": "2026-09-03"}).json()
    assert unchecked["completedDates"] == []

    disabled = client.put(f"/api/daily-checkpoints/{item['id']}", json={"enabled": False}).json()
    assert disabled["enabled"] is False
    response = client.post(f"/api/daily-checkpoints/{item['id']}/toggle", json={"dateKey": "2026-09-03"})
    assert response.status_code == 400

    reenabled = client.put(f"/api/daily-checkpoints/{item['id']}", json={"enabled": True}).json()
    assert reenabled["enabled"] is True
    assert client.delete(f"/api/daily-checkpoints/{item['id']}").status_code == 204


def test_settings_and_data_path_switch(client, tmp_path):
    client.post("/api/ap", json=ap_payload("Persist me"))
    target_dir = tmp_path / "custom-data"
    state = client.get("/api/state").json()
    settings = state["settings"]
    settings["displayName"] = "Minju"
    settings["dataPath"] = str(target_dir)
    saved = client.put("/api/settings", json=settings)
    assert saved.status_code == 200
    actual = target_dir / "eisenhower.json"
    assert saved.json()["dataPath"] == str(actual.resolve())
    assert actual.exists()
    state2 = client.get("/api/state").json()
    assert any(x["title"] == "Persist me" for x in state2["actionPoints"])
