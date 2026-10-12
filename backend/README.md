# EISEN / DAY Go Backend

외부 Go 프레임워크 없이 표준 라이브러리 기반으로 구현한 로컬 REST API입니다.

## 실행

```powershell
go run ./cmd/eisenday
```

기본 bind: `127.0.0.1:8787`

다른 localhost port가 필요하면:

```powershell
$env:EISEN_DAY_ADDR="127.0.0.1:8790"
go run ./cmd/eisenday
```

## 테스트

```powershell
go test ./...
```

## 빌드

```powershell
go build -o eisen-day-backend.exe ./cmd/eisenday
```
