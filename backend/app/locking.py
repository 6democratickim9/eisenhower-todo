from __future__ import annotations

import os
from pathlib import Path
from typing import BinaryIO


class DataFileInUseError(RuntimeError):
    pass


class FileLock:
    """Cross-platform non-blocking lock using a sidecar .lock file.

    Locking a sidecar file lets the JSON itself be atomically replaced without
    losing the lock on Windows or POSIX systems.
    """

    def __init__(self, target: Path):
        self.target = target
        self.path = target.with_suffix(target.suffix + ".lock")
        self.handle: BinaryIO | None = None

    def acquire(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        handle = open(self.path, "a+b")
        try:
            handle.seek(0)
            if handle.tell() == 0:
                handle.write(b"0")
                handle.flush()
            handle.seek(0)
            if os.name == "nt":
                import msvcrt

                try:
                    msvcrt.locking(handle.fileno(), msvcrt.LK_NBLCK, 1)
                except OSError as exc:
                    raise DataFileInUseError(f"Data file is already in use: {self.target}") from exc
            else:
                import fcntl

                try:
                    fcntl.flock(handle.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
                except OSError as exc:
                    raise DataFileInUseError(f"Data file is already in use: {self.target}") from exc
        except Exception:
            handle.close()
            raise
        self.handle = handle

    def release(self) -> None:
        handle = self.handle
        if handle is None:
            return
        try:
            if os.name == "nt":
                import msvcrt

                handle.seek(0)
                try:
                    msvcrt.locking(handle.fileno(), msvcrt.LK_UNLCK, 1)
                except OSError:
                    pass
            else:
                import fcntl

                try:
                    fcntl.flock(handle.fileno(), fcntl.LOCK_UN)
                except OSError:
                    pass
        finally:
            handle.close()
            self.handle = None

    def __enter__(self) -> "FileLock":
        self.acquire()
        return self

    def __exit__(self, *_: object) -> None:
        self.release()
