from __future__ import annotations

import json
import os
import shutil
import tempfile
from pathlib import Path
from typing import Any


def atomic_write_json(path: Path, value: Any, *, backup: bool = True) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8")

    if backup and path.exists():
        try:
            old = path.read_bytes()
            json.loads(old.decode("utf-8"))
            backup_path = path.with_suffix(path.suffix + ".bak")
            shutil.copyfile(path, backup_path)
        except (OSError, UnicodeDecodeError, json.JSONDecodeError):
            pass

    fd, temp_name = tempfile.mkstemp(prefix=path.name + ".", suffix=".tmp", dir=str(path.parent))
    temp_path = Path(temp_name)
    try:
        with os.fdopen(fd, "wb") as handle:
            handle.write(raw)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temp_path, path)
        if os.name != "nt":
            try:
                dir_fd = os.open(path.parent, os.O_DIRECTORY)
                try:
                    os.fsync(dir_fd)
                finally:
                    os.close(dir_fd)
            except OSError:
                pass
    finally:
        if temp_path.exists():
            try:
                temp_path.unlink()
            except OSError:
                pass


def load_json_with_recovery(path: Path) -> tuple[dict[str, Any] | None, Path | None]:
    candidates = [
        path,
        path.with_suffix(path.suffix + ".bak"),
        path.with_suffix(path.suffix + ".old"),
    ]
    candidates.extend(sorted(path.parent.glob(path.name + ".*.tmp"), reverse=True)[:3])
    found = False
    first_error: Exception | None = None
    for candidate in candidates:
        if not candidate.exists():
            continue
        found = True
        try:
            value = json.loads(candidate.read_text(encoding="utf-8"))
            if isinstance(value, dict):
                return value, candidate
            raise ValueError("root JSON value must be an object")
        except (OSError, UnicodeDecodeError, json.JSONDecodeError, ValueError) as exc:
            first_error = first_error or exc
    if found and first_error:
        raise ValueError(f"Planner JSON and recovery files are invalid for {path}: {first_error}")
    return None, None
