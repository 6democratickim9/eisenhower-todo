# EISEN / DAY — Python Local Backend

React/Vite UI를 그대로 사용하면서 로컬 파일과 Jira를 실제로 처리하는 FastAPI 백엔드입니다.

## 구현 범위

- AP CRUD / 완료 / Archive 복원
- Queue 순서 및 모드 저장
- 가장 오래된 미완료 AP를 Queue Priority 1에 고정
- Daily Checkpoint CRUD / Enabled-Disabled / 날짜별 체크 이력
- Jira Watch 저장 / 실제 REST API 동기화 / Done 자동 제거
- AP에 연결된 Jira issue metadata 동기화
- Bearer / Basic Jira 인증, REST API v2/v3
- Jira token은 Planner state와 분리된 `jira-settings.json`에 저장
- Local JSON atomic write + `.bak` recovery
- 별도 `.lock` 파일을 이용한 Windows/POSIX 동시 실행 방지
- 저장 위치 변경 및 기존 JSON 전환
- 기존 Go v1.x JSON (`dailyTasks`, `jiraTickets`) → v6 schema migration
- 기존 `%APPDATA%/EisenhowerTodo` 설정 자동 탐색/마이그레이션
- 127.0.0.1 only bind를 전제로 한 launcher

## 개발 실행

프로젝트 루트에서:

```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements-dev.txt

cd frontend
npm install
$env:VITE_API_MODE='http'
$env:VITE_API_BASE_URL='http://127.0.0.1:8787'
npm run dev
```

다른 터미널에서:

```powershell
.\.venv\Scripts\Activate.ps1
uvicorn backend.app.main:create_app --factory --host 127.0.0.1 --port 8787
```

## 테스트

```powershell
pytest -q backend/tests
```
