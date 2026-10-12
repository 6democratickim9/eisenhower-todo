EISEN / DAY - React + Go 단일 EXE 빌드 버전

이번 UI 업데이트
- Jira Watch 아래에 영구 저장되는 Quick Note 추가
- Q4 명칭을 DON’T DO -> BACKLOG로 변경
- 메인/사분면/사이드바/추가 메뉴의 장식성 그라데이션 제거
- 레퍼런스 이미지처럼 밝고 평평한 카드 + 파스텔 사분면으로 정리
- 담당자/아바타 UI는 추가하지 않음

빌드
1) 최종 EXE는 React production build를 Go binary 안에 embed합니다.
2) Windows에서 build-combined-windows.bat 를 더블클릭하면:
   - npm install/ci
   - React/Vite production build
   - Go tests
   - React dist -> Go embed
   - dist\eisen-day.exe 생성
3) 생성된 eisen-day.exe는 실행할 때 Node/npm/Go가 필요 없습니다.
4) 실행하면 로컬 Go 서버와 React UI가 같이 뜹니다.
5) Windows에서는 Edge app-mode 창으로 열려고 시도하고, 그 창을 닫으면 서버도 종료됩니다.

빌드 시에만 필요한 것: Node.js/npm, Go
