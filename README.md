# EISEN / DAY

**중요도와 긴급도로 업무를 정리하고, 실행 순서부터 일일 점검·Jira·팀 코드리뷰까지 관리하는 로컬 업무 대시보드입니다.**

React + TypeScript 화면과 Go 서버를 하나의 실행 파일로 묶습니다. Windows와 macOS에서 실행하며, 완성된 실행 파일을 사용할 때는 Node.js·Go·Python이나 별도 서버 설치가 필요하지 않습니다.

[Windows / macOS 빌드 결과](https://github.com/6democratickim9/eisenhower-todo/actions/workflows/build-desktop.yml) · [빌드 워크플로](.github/workflows/build-desktop.yml)

## 실행 파일 다운로드

1. 위 빌드 결과 링크에서 성공한 최신 **main** 실행을 엽니다.
2. 페이지의 **Artifacts**에서 운영체제와 CPU에 맞는 파일을 다운로드합니다. GitHub 로그인이 필요할 수 있습니다.
3. 다운로드한 ZIP을 풉니다. Mac은 안에 있는 `tar.gz`도 풀어 주세요.

| 환경 | 아티팩트 | 실행 파일 |
| --- | --- | --- |
| Windows x64 | `EisenDay-windows-x64` | `eisen-day.exe` |
| Mac Apple Silicon (M 시리즈) | `EisenDay-macos-arm64` | `EisenDay-macos` |
| Mac Intel | `EisenDay-macos-x64` | `EisenDay-macos` |

### Windows 실행

`eisen-day.exe`를 더블클릭합니다. 로컬 서버가 시작된 뒤 Microsoft Edge가 있으면 앱 창으로, 없으면 기본 브라우저로 화면을 엽니다.

Edge 앱 창의 프로세스가 종료되면 서버도 종료됩니다. 기존 브라우저 프로세스로 실행이 위임되거나 기본 브라우저를 사용하는 경우에는 창을 닫아도 서버가 남을 수 있습니다. 이때는 작업 관리자에서 해당 `eisen-day.exe` 프로세스를 종료합니다.

### macOS 실행

압축을 푼 폴더에서 터미널로 실행합니다.

```bash
./EisenDay-macos
```

기본 브라우저에 화면이 열립니다. 종료하려면 실행한 터미널에서 **Ctrl+C**를 누릅니다. 브라우저 탭을 닫는 것만으로 서버가 종료되지는 않습니다.

Mac 배포물은 `.app` 번들이 아닌 독립 실행 파일이며 코드 서명·공증을 포함하지 않습니다. macOS가 실행을 차단하면 시스템 설정의 개인정보 보호 및 보안에서 해당 파일의 실행 허용 여부를 확인하세요.

## 주요 기능

| 영역 | 기능 |
| --- | --- |
| Matrix | 중요도·긴급도에 따른 Eisenhower 4분면, 업무 생성·수정·삭제·완료 |
| 업무 유형 | 일반 AP, Meeting, Commit 구분 및 Gerrit 링크 |
| Today's Queue | 수동 순서·중요도·긴급도 기준 실행 순서 |
| 업무 조회 | 전체 업무 검색·필터, 마감일 캘린더, 태그, 통계 |
| Archive | 완료 업무 보관 및 복원 |
| Daily Check | 날짜별 반복 체크포인트, 활성화·비활성화, Jira 연결 |
| Quick Note | 자동 저장되는 개인 메모 |
| Jira | 업무 연결, 상태·담당자 동기화, Jira Watch, 이슈 생성 |
| Team | 공유 폴더에 저장된 팀원 업무 조회 |
| Daily Scrum | 날짜별 완료 업무, 진행 중 Jira, 코드리뷰 현황, 공유 메모 |
| Code Review | 리뷰 요청·응답·승인 조건·MDC 버전·아카이브 관리 |

### Jira 연동

Settings에서 Jira 기본 URL, 인증 방식(Bearer 또는 Basic), API 버전(2 또는 3), 토큰을 설정합니다. Basic 인증에는 사용자 이름도 설정합니다.

- AP에 Jira 티켓을 연결하고 상태·담당자 등을 동기화합니다.
- Jira Watch로 별도 추적할 티켓을 관리합니다.
- AP 생성 시 Jira 이슈 생성 및 관련 이슈 연결을 지원합니다.
- AP 또는 체크포인트 완료 시 Jira 댓글을 남기는 기능은 Settings에서 켜거나 끌 수 있습니다.
- 자동 동기화 주기를 설정할 수 있습니다.

Jira 기능은 설정한 Jira 서버에 연결되어야 동작합니다. 개인 업무 관리 기능은 로컬에서 동작합니다.

### 팀 공유와 코드리뷰

Team 기능은 **로컬에서 접근 가능한 공유 폴더**를 읽고 씁니다. SharePoint를 사용할 때는 동기화된 폴더를 사용하며, SharePoint API에 직접 로그인하는 방식은 아닙니다.

Settings에서 `Team Share Root`와 개인 데이터 경로를 실제 팀 폴더 구조에 맞게 지정합니다. 팀원별 하위 폴더의 업무 데이터를 찾아 표시합니다.

코드리뷰에서는 요청 URL·설명·검증 내용을 기록하고, 두 명의 CR+1 또는 한 명의 CR+2 승인 조건을 선택할 수 있습니다. 완료된 리뷰의 MDC 버전을 기록하고 아카이브할 수 있습니다. Daily Scrum은 팀원별 업무 및 리뷰 현황과 날짜별 공유 메모를 제공합니다.

## 소스에서 빌드

필요 도구:

- Node.js **22 LTS** 및 npm
- Go **1.23 이상**
- 저장소를 clone할 경우 Git

```bash
git clone https://github.com/6democratickim9/eisenhower-todo.git
cd eisenhower-todo
```

### Windows

`build-combined-windows.bat`을 더블클릭하거나 PowerShell에서 실행합니다.

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build-combined-windows.ps1
```

| 결과 | 용도 |
| --- | --- |
| `dist/eisen-day.exe` | 콘솔 창 없이 실행하는 통합 앱 |
| `bin/eisen-day-backend-windows-amd64.exe` | 로그를 확인할 수 있는 콘솔 버전 |

GitHub Actions는 Windows x64에서 빌드합니다. 로컬 스크립트는 설치된 Go의 기본 대상 플랫폼을 사용합니다.

### macOS

```bash
bash ./build-macos.command
./dist/EisenDay-macos
```

Apple Silicon에서는 arm64, Intel Mac에서는 amd64 실행 파일을 생성합니다. `build-macos.command`는 `build-unix.sh`를 호출합니다.

### 빌드 단계

두 OS의 스크립트는 같은 순서로 검증합니다.

1. npm 의존성 설치 (`npm ci`)
2. TypeScript 검사 및 Vite 프로덕션 빌드
3. 새 프론트엔드 결과를 Go의 내장 웹 파일로 복사
4. Go 전체 테스트 (`go test ./...`)
5. 통합 실행 파일 생성

어느 단계에서든 실패하면 빌드를 중단합니다. 실행 파일은 `dist/`에 생성됩니다.

## GitHub Actions

**Build Windows and macOS** 워크플로는 다음 환경에서 각각 테스트하고 빌드합니다.

- Windows x64
- macOS Apple Silicon
- macOS Intel

`main` 및 `build/**` 브랜치 push, PR, `v*` 태그 push에서 자동 실행됩니다. Actions 화면의 **Run workflow**로 직접 실행할 수도 있습니다.

실행 파일은 각 작업의 아티팩트로 저장됩니다. Mac 실행 파일은 실행 권한을 보존하도록 `tar.gz`로 한 번 더 묶습니다. 현재 워크플로는 GitHub Release를 자동 게시하지 않습니다.

## 개발과 테스트

프론트엔드와 Go 서버를 별도 터미널에서 실행할 수 있습니다.

터미널 1 — 백엔드:

```bash
cd backend
go run ./cmd/eisenday
```

터미널 2 — 프론트엔드:

```bash
cd frontend
npm ci
npm run dev
```

개발 화면은 `http://127.0.0.1:5173`에 열립니다. Vite는 `/api` 요청을 `http://127.0.0.1:8787`로 전달합니다. 백엔드 자동 브라우저 실행을 끄려면 실행 전에 `EISEN_DAY_NO_BROWSER=1`을 설정하세요.

개별 검증 명령:

```bash
# 저장소 루트에서
(cd frontend && npm ci && npm run build)
(cd backend && go test ./...)
```

위 괄호 명령은 macOS/bash용입니다. Windows에서는 각 폴더로 이동해 명령을 실행하거나 전체 빌드 스크립트를 사용하세요. 배포 파일을 만들 때는 프론트엔드 내장까지 수행하는 OS별 전체 빌드 스크립트를 사용합니다.

## 데이터와 설정

| 환경 | 기본 설정 폴더 |
| --- | --- |
| Windows | `%APPDATA%/EisenDay` |
| macOS | `~/Library/Application Support/EisenDay` |

설정 폴더에는 다음 파일이 저장됩니다.

- `config.json`: 표시 이름, 데이터 경로 등 설정
- `jira-settings.json`: Jira 인증 설정 및 토큰
- `data/eisenhower.json`: 기본 업무 데이터
- `running.json`: 실행 잠금 정보

Settings에서 업무 데이터 경로를 변경할 수 있습니다. Jira 토큰은 업무 데이터와 분리되지만 로컬 파일에 저장되므로, `jira-settings.json`을 공개 저장소나 공유 자료에 포함하지 마세요.

저장 시 `.bak` 백업을 사용하며, 읽을 때 원본과 `.bak`, `.old`를 차례로 확인합니다. 기존 스키마 변환 경로에서는 `.pre-v6.bak`도 생성합니다. 버전 교체 전에는 설정 폴더와 별도로 지정한 업무 데이터 파일을 백업하세요. Python 버전과의 모든 데이터 호환성을 검증한 것은 아닙니다.

### 환경 변수

| 변수 | 역할 |
| --- | --- |
| `EISEN_DAY_CONFIG_DIR` | 기본 설정 폴더 대신 사용할 경로 |
| `EISEN_DAY_ADDR` | 서버 주소. 기본 `127.0.0.1:8787`, loopback 주소만 허용 |
| `EISEN_DAY_STRICT_ADDR` | 비어 있지 않으면 기본 포트 사용 실패 시 다른 포트로 전환하지 않음 |
| `EISEN_DAY_NO_BROWSER` | 비어 있지 않으면 브라우저 자동 실행 생략 |

기본 포트가 사용 중이면 통합 앱은 빈 로컬 포트로 전환합니다. 개발 모드에서 포트를 바꿀 때는 `frontend/vite.config.ts`의 프록시 대상도 맞춰야 합니다.

## 저장소 구조

```text
backend/
  cmd/eisenday/           실행 진입점과 OS별 브라우저 실행
  internal/planner/      업무 데이터, Jira, 팀 공유, 저장·잠금
  internal/server/       HTTP API
  internal/webui/        Go에 포함되는 프론트엔드
frontend/
  src/                   React 화면과 상태 관리
  public/                정적 이미지
.github/workflows/       Windows·macOS 빌드 자동화
build-combined-windows.* Windows 빌드 진입점
build-macos.command      macOS 빌드 진입점
build-unix.sh            Unix 빌드 공통 처리
```

## 문제 해결

- **`node`, `npm`, `go`를 찾을 수 없음:** 빌드 도구를 설치하고 터미널을 다시 연 뒤 재실행합니다.
- **Mac에서 실행 권한 오류:** 아티팩트의 `tar.gz`까지 풀었는지 확인하고, 필요하면 `chmod +x EisenDay-macos`를 실행합니다.
- **5173 개발 화면에서 API 연결 실패:** Go 서버가 8787에서 실행 중인지 확인합니다. 포트 변경 시 Vite 프록시도 수정합니다.
- **데이터가 이미 사용 중이라는 오류:** 같은 데이터 파일을 사용하는 기존 EisenDay 프로세스를 종료합니다.
- **Jira 동기화 실패:** 기본 URL, 인증 방식, API 버전, 토큰 권한과 서버 연결을 확인합니다.
- **팀원이 표시되지 않음:** 공유 폴더 동기화 상태와 팀원별 데이터 경로를 확인합니다.
