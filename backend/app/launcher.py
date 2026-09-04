from __future__ import annotations

import json
import socket
import sys
import threading
import time
import webbrowser
from pathlib import Path

import uvicorn

from .locking import DataFileInUseError
from .main import create_app
from .paths import resolve_paths
from .persistence import atomic_write_json


class QuietServer(uvicorn.Server):
    def install_signal_handlers(self) -> None:
        return


def pick_socket() -> socket.socket:
    for port in (8787, 0):
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.bind(("127.0.0.1", port))
            sock.listen(2048)
            return sock
        except OSError:
            sock.close()
    raise RuntimeError("사용 가능한 localhost port를 찾을 수 없습니다.")


def _open_existing(running_path: Path) -> bool:
    try:
        data = json.loads(running_path.read_text(encoding="utf-8"))
        url = str(data.get("url") or "")
        if url.startswith("http://127.0.0.1:"):
            webbrowser.open(url)
            return True
    except Exception:
        pass
    return False


def main() -> int:
    paths = resolve_paths()
    try:
        app = create_app(paths)
    except DataFileInUseError:
        if _open_existing(paths.running_path):
            return 0
        print("같은 JSON을 사용하는 EISEN / DAY가 이미 실행 중입니다.")
        return 2

    sock = pick_socket()
    host, port = sock.getsockname()[:2]
    url = f"http://127.0.0.1:{port}"
    atomic_write_json(paths.running_path, {"url": url, "version": "2.0.0"}, backup=False)

    config = uvicorn.Config(app, host="127.0.0.1", port=port, log_level="warning", access_log=False)
    server = QuietServer(config)
    thread = threading.Thread(target=lambda: server.run(sockets=[sock]), name="eisen-day-server", daemon=True)
    thread.start()

    # Wait briefly until the local server is listening.
    time.sleep(0.45)
    print(f"EISEN / DAY v2.0.0\n{url}\nData: {paths.data_path}")

    native_window = False
    try:
        try:
            import webview  # type: ignore

            native_window = True
            webview.create_window(
                "EISEN / DAY",
                url,
                width=1500,
                height=900,
                min_size=(1100, 700),
                background_color="#F3EADC",
            )
            # This call blocks until the desktop window closes.
            webview.start()
            server.should_exit = True
            thread.join(timeout=5)
        except ImportError:
            webbrowser.open(url)
            while thread.is_alive():
                time.sleep(0.4)
    except KeyboardInterrupt:
        server.should_exit = True
        thread.join(timeout=5)
    finally:
        if native_window and thread.is_alive():
            server.should_exit = True
            thread.join(timeout=5)
        try:
            paths.running_path.unlink(missing_ok=True)
        except OSError:
            pass
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
