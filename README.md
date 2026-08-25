# Eisenhower Todo v1.5.1

사내망에서도 사용할 수 있는 **로컬 우선(local-first) Eisenhower Matrix 웹앱**입니다.

## 실행

Windows 배포본에서는 `eisenhower-todo.exe`를 더블클릭합니다.

- Node.js / npm 불필요
- Docker / Python 불필요
- 브라우저 자동 실행
- localhost(`127.0.0.1`)에만 바인딩

## v1.4.0 Dashboard

레퍼런스 UI를 기준으로 대시보드를 재구성했습니다.

- 중앙: Eisenhower Q1~Q4 Matrix
- 하단: Today's Queue / Calendar / Tags
- 우측: 상시 노출 AP 추가·편집 폼 / Statistics
- Archive는 하단 별도 영역 유지
- 상단/사분면의 `AP 추가` 버튼 제거
- 카드의 `편집`을 누르면 우측 AP 폼에서 바로 수정
- Tags를 AP 데이터 모델에 추가하고 쉼표 입력 지원
- Tags 패널에서 태그별 개수 표시 및 클릭 필터
- Calendar는 AP Deadline을 월 단위로 표시
- Statistics는 Q1~Q4 분포와 최근 7일 완료 추이를 표시

## AP

- Title: 필수
- Details: 선택
- Importance: 0~100 직접 설정
- Base Urgency: 0~100 직접 설정
- Deadline: 선택
- Tags: 선택
- Jira: 선택

Deadline이 가까워지면 Effective Urgency가 자동 상승하며 Q2→Q1, Q4→Q3로 이동할 수 있습니다.

## JSON 저장 위치

첫 실행 시 `eisenhower.json` 위치를 확인하거나 선택할 수 있습니다.

- Windows 폴더 선택 지원
- 직접 경로 입력 지원
- 기존 JSON을 지정하면 기존 데이터를 로드
- 새 경로면 현재 데이터를 새 위치에 저장
- 선택 위치는 exe 옆 `eisenhower-config.json`에 기억
- 백업은 선택한 JSON 옆 `<filename>.bak`

## Jira

Jira 설정은 exe 옆 `data/jira-settings.json`에 별도 저장합니다.

지원 설정:

- Jira Base URL
- Token 발급 페이지 URL
- Bearer 또는 Basic 인증
- Username/Email (Basic)
- Token
- Jira REST API v2/v3
- 자동 Sync 주기

Jira 연결 AP에서 표시:

- Status
- Assignee
- 최근 댓글 1개
- Last Sync
- Sync 오류

긴 Jira 댓글은 기본 2줄만 표시하고 `더보기`로 펼칠 수 있습니다. Jira Cloud ADF 댓글도 paragraph/hard break를 가능한 범위에서 줄바꿈으로 보존합니다.

Jira fetch 시점:

- 앱 시작
- AP 생성/수정/이동/삭제
- 완료/복원
- Queue 변경/재정렬
- Jira 설정 저장
- 수동 Jira Sync
- 자동 Sync 주기

Jira Status/Assignee/최근 댓글 변경은 AP History에 `jira-change`로 기록합니다.

## 설정 Dialog

v1.3.0에서 발생했던 Jira 설정 / 저장 위치 Dialog의 `×`가 동작하지 않는 문제를 수정했습니다.

- 모든 이벤트 바인딩을 null-safe 방식으로 변경
- Jira 설정 `×` / 취소 정상 닫힘
- 저장 위치 `×` / 취소 정상 닫힘
- backdrop 클릭으로도 닫힘
- 첫 실행 저장 위치 안내를 닫으면 현재 세션에서는 반복해서 다시 열리지 않음

## 소스 빌드

```bash
go build -o eisenhower-todo .
```

Windows amd64:

```bash
GOOS=windows GOARCH=amd64 go build -o eisenhower-todo.exe .
```


## v1.5.0 Icon polish

- Replaced pseudo-glyph/CSS-looking icons with native emoji icons.
- No external icon CDN or image dependency, so the UI stays fully offline-friendly.
- Added a local emoji favicon.
- Updated sidebar, summary cards, quadrants, Jira link, deadline and complete icons.


## v1.5.0 storage reliability

This release changes persistence to avoid apparent data resets after upgrades.

### Stable config location

The selected planner JSON path is now remembered in the user's OS config directory:

- Windows: `%APPDATA%\EisenhowerTodo\config.json`
- Jira settings: `%APPDATA%\EisenhowerTodo\jira-settings.json`
- Default planner data: `%APPDATA%\EisenhowerTodo\data\eisenhower.json`

Moving or upgrading `eisenhower-todo.exe` no longer creates a new independent configuration just because the EXE lives in a different folder.

Existing v1.2-v1.4.1 portable config is migrated automatically when it is beside the upgraded EXE.

### Safer writes

Planner data is written to `.tmp`, flushed to disk, and then replaced safely.

On Windows the previous main file is rotated to `.old` before replacement. If the replacement fails, the old file is restored instead of deleting the main JSON first.

The app also maintains:

- `<planner>.bak`
- `<planner>.tmp` during writes
- `<planner>.old` only during Windows replacement

### Automatic recovery

At startup, if the main JSON is missing or invalid, the app tries valid recovery copies in this order:

1. main JSON
2. `.tmp`
3. `.old`
4. `.bak`

It will not silently initialize an empty planner when a valid recovery file is available.


## v1.5.0 Jira issue-key linking

The AP editor now asks only for an Issue Key.

- `ABC-123` is enough.
- When Jira Base URL is `https://jira.company.internal`, the ticket URL becomes:
  `https://jira.company.internal/browse/ABC-123`
- Existing JSON/API data that contains only a ticket URL is still supported: the issue key is extracted from the URL.
- Either `issueKey` or `url` can identify a Jira ticket in stored/imported data.
- If both exist, the configured Jira Base URL + Issue Key is used as the canonical ticket URL.


## v1.5.0 concurrent-writer protection

Running multiple copies against the same planner JSON could previously cause last-writer-wins data loss.

Example:

- instance A loads A/B/C
- instance B loads A/B/C
- A adds D and saves A/B/C/D
- B adds E from stale memory and saves A/B/C/E

D disappears.

### Windows protection

v1.5.0 opens the active planner JSON as an exclusive writer for the lifetime of the process.

- other processes can read the JSON
- another Eisenhower Todo process cannot open the same JSON for writing
- older versions attempting to overwrite or rename the same JSON are blocked by Windows while v1.5.0 owns it
- a second v1.5.0 instance exits instead of becoming another writer

v1.5.0 writes through the locked handle and keeps a valid `.bak` snapshot before each write. If a write fails, it rolls the main file back to its previous contents.

The active instance also records its localhost URL in `%APPDATA%\EisenhowerTodo\running.json`, so a second v1.5.0 launch can reopen the already-running instance.


## v1.5.0

### Direct JSON writes
Normal saves no longer create `.tmp` or `.old`.
The active planner JSON is updated directly on every change.
A `.bak` snapshot is kept for recovery.

### Jira HTML/SSO diagnosis
The app no longer follows Jira redirects automatically.
302/3xx, HTML login pages, 401, 403 and 404 are reported separately instead of showing `invalid character '<'`.
The connection test uses `/myself` when no linked issue exists.


## v1.5.0 version collision handling

If an older Eisenhower Todo version still owns the planner JSON lock, the new executable no longer silently reopens that old browser session.

- same v1.5.0 already running: reopen the existing v1.5.0 page
- older/different version running: show a Windows warning and exit
- UI always displays the backend version badge

This prevents users from thinking they are testing a new Jira parser while an older process is actually serving the page.


## v1.5.0 per-user display name

The greeting name is no longer hard-coded.

- Click the name / pencil in the header, or use `이름 설정` in the sidebar.
- The display name is stored in the per-user app config, not in the planner JSON.
- Sharing the executable with coworkers does not make everyone share the same name.
- Users can point at the same planner JSON while keeping independent local display names.
- Existing users upgrading from the older hard-coded greeting keep `Minju` until they change it; fresh installs default to the OS account name.


## v1.5.0 active-only + AP aging
- unfinished APs persist until completed/deleted
- completed APs are removed instead of archived
- existing Archive entries are cleared on first v1.5.0 load
- cards and queue show `N일`
- age colors use both absolute age and relative age among active APs
- Statistics shows active AP aging bands instead of 7-day completion trend


## v1.5.0 visual redesign
- custom local SVG icon system and botanical illustration
- palette: #1B4436 / #88A33B / #F9E4C5 / #DDAD8A / #5B758C
- warm editorial / catalog-inspired card design
- no external CDN, font, or image dependency
- AP age badge shortened to `N일`
- v1.4.8 functionality preserved


## v1.5.1
- Final confirmed warm editorial UI implementation.
- 2x2 Matrix and four bottom widgets preserved on desktop layouts.
- Local decorative PNG assets added under `web/assets`.
- Oldest active AP is pinned as Today's Queue Priority 1 without changing its quadrant.
- Jira recent comment is clamped to two lines with expand/collapse.
- AP editor provides Q1-Q4 presets while keeping Importance/Base Urgency sliders editable.
