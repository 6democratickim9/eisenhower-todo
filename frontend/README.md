# EISEN / DAY — React + TypeScript + Vite UI

승인된 디자인을 유지한 프론트엔드입니다. Production build는 Python/FastAPI local backend와 **same-origin**으로 통신합니다.

## 디자인 고정

- Deep Forest sidebar / warm neutral canvas
- Q1 Coral / Q2 Mint / Q3 Lavender / Q4 Butter
- Home desktop: 2×2 Matrix + Right Rail
- AP/Daily editor: right overlay drawer
- 외부 CDN/Google Fonts 없음

## API mode

개발 중 frontend-only 확인:

```env
VITE_API_MODE=local
```

FastAPI와 분리 개발:

```env
VITE_API_MODE=http
VITE_API_BASE_URL=http://127.0.0.1:8787
```

Production `.env.production`은 `VITE_API_MODE=http`, 빈 API base를 사용합니다. 따라서 packaged EXE가 8787 대신 동적 localhost port를 사용해도 자동으로 같은 origin의 `/api/...`에 연결됩니다.

## 기능

- AP CRUD / complete / edit drawer
- Daily Checkpoint CRUD / enable-disable / 날짜별 toggle
- Matrix Drag & Drop
- Today’s Queue reorder / mode
- Jira Watch + backend sync
- Settings + Jira auth/token test
- Calendar / Tags / Statistics
- Local mode에서는 localStorage mock adapter

## build

```bash
npm install
npm run build
```

Windows 최종 package는 프로젝트 루트의 `build_windows.ps1`을 사용합니다.
