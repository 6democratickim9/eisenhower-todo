# Release packaging

Source and end-user packages are intentionally separate.

- `EisenDay-source.zip`: developer package. Contains `backend/`, `frontend/`, tests, and build scripts.
- `release/EisenDay-windows-portable.zip`: generated on Windows and contains only `EisenDay.exe`.
- `release/EisenDay-macos-<arch>.zip`: generated on macOS and contains only `EisenDay.app`.

Build on the target OS:

```powershell
.\build_windows.ps1
```

```bash
chmod +x build_macos.sh
./build_macos.sh
```

Do not distribute the source ZIP as the normal end-user download.
