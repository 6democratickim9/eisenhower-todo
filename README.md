# EISEN / DAY — Personal Ops Dashboard

## Features

### Core
- Eisenhower 4분면 Matrix (Q1~Q4) — AP 생성/편집/삭제/완료
- AP 종류 선택 — AP / Meeting / Commit
- Commit AP — Gerrit URL 입력, 완료 시 자동 Code Review 요청 생성 (Code Review History에서 수정 가능)
- Matrix AP 카드에서 Jira 티켓 번호 클릭 시 Jira 페이지 이동
- Today's Queue — 실행 순서 관리 (manual/importance/urgency 정렬)
- All AP — 전체 AP 검색/필터
- Calendar — deadline 월간 뷰
- Tags — 컨텍스트별 AP 필터
- Statistics — Quadrant 분포, workload 지표
- Archive — 완료 AP 조회 + 복구 (Restore)
- Quick Note — 메모장 (자동 저장)
- Deadline → baseUrgency 서버 자동 증가 (24h 이내 +8, 72h +4, 168h +1, 상한 100)

### Jira Integration
- AP에 Jira 티켓 연결 (issueKey, status, assignee 자동 sync)
- AP 완료 시 Jira에 영어 댓글 자동 작성 (Settings on/off, details 포함 선택)
- AP 생성 시 Jira 티켓 자동 생성 (project key, issue type, linked issue)
- Jira Watch — 개인 추적 티켓 목록 (수동 추가, 자동 sync)
- Jira Config — baseUrl, authMode (bearer/basic), apiVersion (2/3)

### Daily Check
- 매일 반복 체크포인트 (enable/disable/toggle)
- Checkpoint에 Jira 티켓 연결
- Checkpoint 완료 시 Jira에 영어 댓글 자동 작성 (Settings on/off)

### Team (SharePoint 동기화)
- Team AP — 팀원별 active AP 읽기 전용 조회
- 팀 공유 폴더 기반 자동 감지 (Settings에서 root 설정)

### Daily Scrum
- 팀원별 오늘 완료 AP 조회
- 팀원별 진행중 Jira 티켓 표시 (JiraWatch 기반)
- Jira 티켓 상태 라벨: ⏸ 중단 (수동 paused) / ⚠ N일 정체 (5일 이상 업데이트 없음)
- 해당 날짜에 complete된 코드리뷰 현황 표시 (MDC 기입됨/미입력 구분)
- 날짜 탐색 (‹ ›)
- Scrum Note — 날짜별 공유 메모 (자동 저장)

### Code Review
- 코드리뷰 요청 생성 (URL, title, description, validation)
- 코드리뷰 요청 수정 (요청자 본인만, 인라인 편집)
- 사이드바 리뷰 카운트 + 형광 pulse 알림 (active만 카운트)
- 메인 상단 배너 알림 (incoming review / completed review, archived 제외)
- Approval mode 선택 — "각자 CR+1" (2명 필요) / "CR+2 한명" (1명이면 complete)
- 리뷰 응답 — 완료 / 내 영역 아님
- MDC 버전 기입 — 요청자가 리뷰 complete 후 RCC-MDC, RGW-MDC, COMMON-MDC 3개 기입
- 수동 아카이브 — complete된 리뷰를 📦 Archive 버튼으로 아카이브 (소유자 또는 리뷰어)
- MDC 기입 완료 시 자동 아카이브 (Active에서 숨김, Archived 탭에서 조회)
- 필터 — Active / All / My requests / To review / Archived

### Infrastructure
- Go backend + React frontend (Vite) — 단일 .exe로 패키징
- Edge app mode 실행 (고정 프로필 경로 재사용)
- 서버 ready 대기 후 브라우저 실행 (첫 실행 connection failed 방지)
- 프론트엔드 API 재시도 (최대 10회, 500ms 간격)
- CORS localhost 허용
- Auto Jira sync (설정 간격)

## Windows / macOS 빌드

Node.js 22 LTS(npm 포함), Go 1.23 이상을 설치한 뒤 빌드합니다.
실행 파일에 웹 화면과 서버가 포함되므로 실행 시에는 Node.js나 Go가 필요하지 않습니다.

### Windows

`build-combined-windows.bat`을 더블클릭합니다. 또는 PowerShell에서:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build-combined-windows.ps1
```

결과: `dist/eisen-day.exe`. 더블클릭하면 서버와 브라우저가 실행됩니다.
진단용 콘솔 실행 파일도 `bin/`에 생성됩니다.

### macOS (Apple Silicon / Intel)

터미널에서 저장소 폴더로 이동한 뒤:

```bash
bash ./build-macos.command
./dist/EisenDay-macos
```

현재 Mac의 CPU에 맞춰 빌드합니다. 기본 브라우저가 열리며 터미널에서 Ctrl+C로 종료합니다.
결과는 `.app` 번들이 아닌 독립 실행 파일입니다. 코드 서명과 공증은 포함하지 않습니다.

### GitHub Actions

**Build Windows and macOS** 워크플로에서 Windows x64, Mac Apple Silicon, Mac Intel을 각각 빌드합니다.
`main`, `build/**` 브랜치 push와 PR에서 자동 실행됩니다. 기본 브랜치에 병합한 후 수동 실행도 가능합니다.
각 OS에서 프론트엔드 빌드, Go 전체 테스트, 실행 파일 생성을 수행합니다.

Artifacts에서 해당 OS 파일을 다운로드합니다. Mac은 ZIP 안의 tar.gz도 풀어 실행 권한을 보존합니다.

### 검증

전체 빌드 스크립트가 `npm ci`, `npm run build`, `go test ./...`, `go build`를 차례로 실행합니다.
