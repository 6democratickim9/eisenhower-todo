from __future__ import annotations

import os
import threading
import time
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any

from fastapi import FastAPI, HTTPException, Response
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles

from .jira import JiraError
from .locking import DataFileInUseError
from .models import (
    APDraft,
    ActionPoint,
    DailyCheckpoint,
    DailyCheckpointDraft,
    DailyToggleRequest,
    JiraConfigPublic,
    JiraConfigUpdate,
    JiraWatchItem,
    JiraWatchSaveRequest,
    PlannerSettings,
    PlannerState,
    QueueRequest,
)
from .paths import ResolvedPaths, resolve_paths
from .store import PlannerStore


def _dump(model: Any) -> Any:
    if hasattr(model, "model_dump"):
        return model.model_dump(by_alias=True, exclude_none=True)
    if isinstance(model, list):
        return [_dump(x) for x in model]
    return model


def _frontend_dist() -> Path | None:
    explicit = os.environ.get("EISEN_DAY_FRONTEND_DIST")
    candidates: list[Path] = []
    if explicit:
        candidates.append(Path(explicit))
    # Source tree: backend/app/main.py -> project/frontend/dist
    candidates.append(Path(__file__).resolve().parents[2] / "frontend" / "dist")
    # PyInstaller extraction directory.
    try:
        import sys

        if getattr(sys, "_MEIPASS", None):
            candidates.append(Path(sys._MEIPASS) / "frontend" / "dist")  # type: ignore[attr-defined]
    except Exception:
        pass
    for candidate in candidates:
        if (candidate / "index.html").exists():
            return candidate
    return None


def create_app(paths: ResolvedPaths | None = None, *, start_auto_sync: bool = True) -> FastAPI:
    paths = paths or resolve_paths()
    store = PlannerStore(paths)
    stop_event = threading.Event()

    def auto_sync_loop() -> None:
        # Initial delay prevents startup from blocking on an unreachable Jira.
        stop_event.wait(3)
        while not stop_event.is_set():
            try:
                state = store.state()
                interval = max(1, state.settings.auto_sync_minutes) * 60
                if store.jira_client.configured(store.jira_config) and state.jira_watch:
                    store.sync_jira_watch()
            except Exception:
                pass
            stop_event.wait(interval if "interval" in locals() else 900)

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        thread: threading.Thread | None = None
        if start_auto_sync:
            thread = threading.Thread(target=auto_sync_loop, name="jira-auto-sync", daemon=True)
            thread.start()
        try:
            yield
        finally:
            stop_event.set()
            if thread:
                thread.join(timeout=1)
            store.close()

    app = FastAPI(title="EISEN / DAY Local API", version="2.0.0", docs_url=None, redoc_url=None, lifespan=lifespan)
    app.state.store = store

    app.add_middleware(
        CORSMiddleware,
        allow_origin_regex=r"^https?://(127\.0\.0\.1|localhost)(:\d+)?$",
        allow_credentials=False,
        allow_methods=["GET", "POST", "PUT", "DELETE", "OPTIONS"],
        allow_headers=["Content-Type"],
    )

    @app.middleware("http")
    async def security_headers(request, call_next):
        response = await call_next(request)
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["X-Frame-Options"] = "DENY"
        response.headers["Referrer-Policy"] = "no-referrer"
        response.headers["Cache-Control"] = "no-store" if request.url.path.startswith("/api/") else "no-cache"
        return response

    def fail(exc: Exception) -> HTTPException:
        if isinstance(exc, KeyError):
            return HTTPException(status_code=404, detail=str(exc).strip("'"))
        if isinstance(exc, (ValueError, JiraError)):
            return HTTPException(status_code=400, detail=str(exc))
        return HTTPException(status_code=500, detail=str(exc))

    @app.get("/api/state", response_model=PlannerState, response_model_by_alias=True)
    def get_state():
        return store.state()

    @app.post("/api/ap", response_model=ActionPoint, status_code=201, response_model_by_alias=True)
    def create_ap(draft: APDraft):
        try:
            return store.create_ap(draft)
        except Exception as exc:
            raise fail(exc) from exc

    @app.put("/api/ap/{ap_id}", response_model=ActionPoint, response_model_by_alias=True)
    def update_ap(ap_id: str, patch: dict[str, Any]):
        try:
            return store.update_ap(ap_id, patch)
        except Exception as exc:
            raise fail(exc) from exc

    @app.delete("/api/ap/{ap_id}", status_code=204)
    def delete_ap(ap_id: str):
        try:
            store.delete_ap(ap_id)
            return Response(status_code=204)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/ap/{ap_id}/complete", status_code=204)
    def complete_ap(ap_id: str):
        try:
            store.complete_ap(ap_id)
            return Response(status_code=204)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/archive/{ap_id}/restore", status_code=204)
    def restore_ap(ap_id: str):
        try:
            store.restore_ap(ap_id)
            return Response(status_code=204)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/queue", status_code=204)
    def reorder_queue(request: QueueRequest):
        try:
            store.reorder_queue(request.ids, request.mode)
            return Response(status_code=204)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/daily-checkpoints", response_model=DailyCheckpoint, status_code=201, response_model_by_alias=True)
    def create_daily(draft: DailyCheckpointDraft):
        try:
            return store.create_daily(draft)
        except Exception as exc:
            raise fail(exc) from exc

    @app.put("/api/daily-checkpoints/{item_id}", response_model=DailyCheckpoint, response_model_by_alias=True)
    def update_daily(item_id: str, patch: dict[str, Any]):
        try:
            return store.update_daily(item_id, patch)
        except Exception as exc:
            raise fail(exc) from exc

    @app.delete("/api/daily-checkpoints/{item_id}", status_code=204)
    def delete_daily(item_id: str):
        try:
            store.delete_daily(item_id)
            return Response(status_code=204)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/daily-checkpoints/{item_id}/toggle", response_model=DailyCheckpoint, response_model_by_alias=True)
    def toggle_daily(item_id: str, request: DailyToggleRequest):
        try:
            return store.toggle_daily(item_id, request.date_key)
        except Exception as exc:
            raise fail(exc) from exc

    @app.put("/api/jira/watch", status_code=204)
    def save_watch(request: JiraWatchSaveRequest):
        try:
            store.save_jira_watch(request.items)
            return Response(status_code=204)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/jira/watch/sync", response_model=list[JiraWatchItem], response_model_by_alias=True)
    def sync_watch():
        try:
            return store.sync_jira_watch()
        except Exception as exc:
            raise fail(exc) from exc

    @app.get("/api/jira/config", response_model=JiraConfigPublic, response_model_by_alias=True)
    def jira_config():
        return store.jira_public_config()

    @app.put("/api/jira/config", response_model=JiraConfigPublic, response_model_by_alias=True)
    def save_jira_config(patch: JiraConfigUpdate):
        try:
            return store.save_jira_config(patch)
        except Exception as exc:
            raise fail(exc) from exc

    @app.post("/api/jira/test")
    def test_jira():
        try:
            return {"ok": True, "message": store.test_jira()}
        except Exception as exc:
            raise fail(exc) from exc

    @app.put("/api/settings", response_model=PlannerSettings, response_model_by_alias=True)
    def save_settings(settings: PlannerSettings):
        try:
            return store.save_settings(settings)
        except Exception as exc:
            raise fail(exc) from exc

    @app.get("/api/info")
    def info():
        state = store.state()
        return {
            "version": "2.0.0",
            "dataPath": state.settings.data_path,
            "configPath": str(store.paths.config_path),
            "jiraSettingsPath": str(store.paths.jira_path),
            "frontendBuilt": _frontend_dist() is not None,
        }

    dist = _frontend_dist()
    if dist:
        assets = dist / "assets"
        if assets.exists():
            app.mount("/assets", StaticFiles(directory=str(assets)), name="assets")

        @app.get("/{full_path:path}", include_in_schema=False)
        def frontend(full_path: str):
            requested = (dist / full_path).resolve() if full_path else dist / "index.html"
            try:
                requested.relative_to(dist.resolve())
            except ValueError:
                raise HTTPException(status_code=404)
            if full_path and requested.is_file():
                return FileResponse(requested)
            return FileResponse(dist / "index.html")
    else:
        @app.get("/", include_in_schema=False)
        def frontend_missing():
            return {
                "ok": True,
                "message": "Backend is running. Build frontend with `npm run build` before packaged use.",
            }

    return app


# Development: `uvicorn backend.app.main:create_app --factory`
