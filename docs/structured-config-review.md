# 구조화 설정 자체 코드 리뷰 및 릴리스 검증

검증일: 2026-11-11. 사용자 승인에 따라 v1.4.0 / v2.1.0 정식 릴리스를 진행한다.
아래는 기능·회귀 검증 기록이며 실제 발행 여부는 원격 태그와 GitHub Release를 기준으로 확인한다.

## 기능 개발 브랜치

| 대상 | 브랜치 | 기준 | 작업 디렉터리 |
|---|---|---|---|
| v2.1.0 | feat/yaml-structured-config | v2.0.0 / 032c181 | fatima-core |
| v1.4.0 | feat/yaml-structured-config-v1 | v1.3.7 / ad4d910 | ../fatima-core-v1-structured |

v2의 기능 커밋은 `28fb9ac`, v1의 동일 기능 backport는 `32f66ce`이며
v1 import 경로 보정은 `8eaed02`다. 양쪽 기능 변경 17개 파일은 모듈 경로를 정규화하면 동일하다.
v1의 opm, go.mod, go.sum은 v1.3.7에서 변경하지 않았다. 기존 릴리스 태그는 유지한다.

## 자체 리뷰 범위와 수정 사항

| 검토 항목 | 결과 |
|---|---|
| 병합 순서 | 파일/문서 선택 → 객체 재귀 병합/배열 교체 → 문자열 해석 → 두 조회 방식 제공 |
| [] / null / 타입 교체 | base 값과 이전 하위 문자열 키, stale 목록 메타데이터가 남지 않음 |
| 문자열·타입 | 변수 치환 결과를 다시 YAML로 파싱하지 않음. strict decode와 정수 overflow 검증 |
| 스냅샷 | LookupNode/DecodeNode의 map·slice가 내부 데이터와 공유되지 않음. 실패 시 목적지 불변 |
| API 호환 | 원래 5개 메서드만 구현한 Config도 컴파일 가능. 새 인터페이스는 선택적 |
| 실제 runtime wiring | NewPropertyConfigReader → DefaultProcessBuilder.GetConfig → FatimaRuntimeProcess.GetConfig 점검. 실제 getter 경로 테스트 |
| mock wiring | NewMockConfig의 resources 로더 및 NewMockConfigWithMap에 새 조회 연결. 전역 mock runtime 연결/복구 테스트 |
| 숫자 YAML 키 | 리뷰 중 루트 숫자 키의 암묵 문자열 변환 가능성을 발견. any로 먼저 읽고 문자열 map만 허용하도록 수정 |
| secret/오류 로그 | 기존 기본 secret 해석기와 GetInt 오류의 값 노출 제거. 반환 정책 유지. 로그 유출 회귀 테스트 |
| 모듈 경로 | v1 신규 파일의 /v2 import를 제거. 양쪽 기능과 fixture 일치 검사 |

## 실행 결과

| 검사 | v2 | v1 |
|---|---|---|
| go test ./builder -count=1 (전체 검사에도 포함) | 통과 | 통과 |
| go test -race ./builder ./runtime ./crypt -count=1 | 통과 | 통과 |
| go build ./... | 통과 | 통과 |
| go test ./... -count=1 | 기존 IPC 테스트에서 실패 | 기존 IPC 테스트에서 실패 |
| go vet ./... | 기존 copylocks 경고 2건 | 기존 copylocks 경고 2건 |
| go vet -copylocks=false ./... | 통과 | 통과 |
| git diff --check | 통과 | 통과 |

전체 테스트 실행에서 IPC 이외 패키지는 통과했다. v1의 기존 opm 패키지 테스트도 통과했다.
v2 IPC의 독립 TestTransaction도 통과했다. 전체 검사 통과로 보고하지 않는다.

### 변경 전부터 존재하는 검증 한계

- `ipc.TestCronExecuteOnyJobName`: fatimaRuntime이 초기화되지 않은 테스트에서
  `ipc_env.go:58` → `buildSockDir`의 nil pointer panic. v2 수정 전과 v1.3.7 원본에서 재현했다.
  기존 RELEASE.md v1.3.5에도 기록되어 있다. 이 panic으로 IPC 패키지의 후속 테스트는 완료되지 않는다.
- `builder/notify_grpc.go:99,110`: protobuf 메시지를 값으로 넘기는 기존 코드의 copylocks 경고.
  v1 원본에서도 동일하게 재현했고 해당 코드는 이번 변경에서 수정하지 않았다.

## 검수할 동작 변경

[설정 사용 가이드](structured-config.md)에 상세 예제와 신규 규칙이 있다.
특히 `[]`를 명시하면 기존 getter에서도 비우고, null이면 문자열 조회에서 제거한다.
명시하지 않은 키는 상속한다. 점 표기 충돌은 결정적인 순서를 사용하며 YAML secret 배열은 요소별로 해석한다.
be-knowledge-svc의 별도 파서 제거와 등록 정책 이관은 이번 변경에 포함하지 않는다.

## 로딩 상태 보완 — 1단계 완료

knowledge-svc 이관 검토에서 발견한 오류 전달 공백을 두 준비 브랜치에 보완했다.
서비스 수정·통합 검증은 별도 후속 단계이며 이번 보완으로 완료되었다고 판단하지 않는다.

- 기존 `Config`와 `StructuredConfig`를 변경하지 않고 선택적 `ConfigLoadStatus`를 추가했다.
  `LoadError`는 파일 읽기/파싱 실패 전체를, `HasBaseConfig`는 base 파일의 로딩 성공 여부를 제공한다.
- 기본 WARN 후 계속 진행 정책, 프로필 전용 파일 허용, 없는 선택적 프로필 파일 허용을 유지한다.
  base 필수 정책과 기동 중단은 소비 서비스가 선택한다. 변수·secret·업무 검증은 이 상태에 포함하지 않는다.
- 실제 `NewPropertyConfigReader → DefaultProcessBuilder → FatimaRuntimeProcess.GetConfig`,
  전역 runtime getter, resources 기반 mock, map 기반 mock과 설정 없는 mock을 점검했다.
  임베딩된 ConfigSnapshot을 통해 같은 상태를 제공하며 런타임 테스트로 오류 전달을 검증했다.
- 자체 리뷰에서 `.properties`의 `Scanner.Err()` 누락을 발견하고 수정했다.
  application 로더는 실패한 파일의 부분값을 버리고 오류를 보고한다.
  공용 읽기 함수는 부분값과 오류를 함께 반환하므로 기존 predefine 호출자의 부분값 사용은 유지한다.
- 반환 오류는 파일명·고정 오류 종류만 담는다. 여러 실패를 보존하고, 반환 진단을 변경해도
  스냅샷은 변하지 않음을 검증했다. 디스크 수정으로 기존 스냅샷의 상태가 바뀌지 않는다.

추가 검증: 양쪽에서 `go test -race ./builder ./runtime ./crypt -count=1`,
`go build ./...`, `go vet -copylocks=false ./...`, `git diff --check` 통과.
정상/빈 base, 설정 없음, profile 전용, 선택적 profile 없음, base/profile 각각 및 동시 오류,
다중 문서의 무시된 별도 파일, 비선택 문서의 문법 오류, 파일 읽기 실패와 properties 스캔 오류를 검증했다.
정식 릴리스 검증에서 전체 테스트와 기본 vet도 다시 실행했다. 결과는 위 기록과 동일하다.
전체 테스트의 기존 IPC panic과 기본 vet의 copylocks 경고 2건은 남아 있으며 전체 검사 통과로 보고하지 않는다.

## 정식 릴리스 기준

- v1.4.0: v1.3.7에서 분기한 유지보수 브랜치 `release/1.x`. v1 모듈 경로와 opm을 유지한다.
- v2.1.0: `main`. `/v2` 모듈 경로를 유지한다.
- 기존 태그는 변경하지 않고 각 릴리스 커밋에 새 annotated tag를 발행한다.
- main의 보호 규칙을 따라 PR로 변경을 반영한다.
- Go 모듈 다운로드 후 태그 커밋·모듈 경로를 확인한다.
- knowledge-svc의 의존성은 정식 v1.4.0으로 전환하며 서비스 빌드·배포는 사용자가 수행한다.

## QA 의존 버전 교체 확인의 범위

knowledge-svc에서 개발 pseudo-version과 사용자 롤백 후 v1.3.7을 비교했다.
대상 목록·기능 목록·검색은 양쪽 모두 HTTP 200, 반환 샘플 항목이 같았고 약 20초 지연도 공통으로 재현됐다.
이 조회 지연은 별도 조사 대상이며 이번 core 릴리스의 수정 범위에 포함하지 않는다.
qa는 API 전용 모드였고 별도 YAML 등록 로더를 구조화 API로 교체하지 않았으므로,
이번 결과를 서비스 로더 통합 완료나 전체 운영 기능 검증으로 해석하지 않는다.
