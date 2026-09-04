# -*- mode: python ; coding: utf-8 -*-
import os
from pathlib import Path
from PyInstaller.utils.hooks import collect_submodules

root = Path(SPECPATH)
frontend_dist = root / 'frontend' / 'dist'
if not (frontend_dist / 'index.html').exists():
    raise RuntimeError('frontend/dist is missing. Run `npm run build` first.')

# Build the architecture of the Python interpreter by default.
# Optional override: EISEN_DAY_TARGET_ARCH=arm64|x86_64|universal2
# universal2 only works when the Python installation and collected native
# dependencies also provide universal2 binaries.
target_arch = os.environ.get('EISEN_DAY_TARGET_ARCH') or None
codesign_identity = os.environ.get('EISEN_DAY_CODESIGN_IDENTITY') or None

hiddenimports = []
hiddenimports += collect_submodules('uvicorn')
hiddenimports += collect_submodules('webview')

analysis = Analysis(
    [str(root / 'launcher.py')],
    pathex=[str(root)],
    binaries=[],
    datas=[(str(frontend_dist), 'frontend/dist')],
    hiddenimports=hiddenimports,
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[],
    noarchive=False,
    optimize=0,
)
pyz = PYZ(analysis.pure)

# onedir + windowed is preferred for macOS app bundles; it avoids the
# unpack-on-every-launch penalty of onefile .app bundles.
exe = EXE(
    pyz,
    analysis.scripts,
    [],
    exclude_binaries=True,
    name='EisenDay',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=False,
    console=False,
    disable_windowed_traceback=False,
    argv_emulation=False,
    target_arch=target_arch,
    codesign_identity=codesign_identity,
    entitlements_file=None,
)

coll = COLLECT(
    exe,
    analysis.binaries,
    analysis.datas,
    strip=False,
    upx=False,
    upx_exclude=[],
    name='EisenDay',
)

app = BUNDLE(
    coll,
    name='EisenDay.app',
    icon=None,
    bundle_identifier='app.eisenday.desktop',
    version='2.0.0',
    info_plist={
        'CFBundleDisplayName': 'EISEN / DAY',
        'CFBundleName': 'EisenDay',
        'NSHighResolutionCapable': True,
    },
)
