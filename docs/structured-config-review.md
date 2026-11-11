# 구조화 설정 자체 코드 리뷰 및 릴리스 준비

검증일: 2026-11-11. **사용자 최종 검수 대기. 태그 생성·푸시·릴리스 게시를 수행하지 않음.**

## 준비한 브랜치

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

## 승인 후 릴리스 단계 (미실행)

사용자가 최종 검수 후 명시적으로 요청하면, 검수된 커밋의 브랜치 반영과
v1.4.0 / v2.1.0 태그 생성·원격 게시를 진행한다. 그 전에 위 기존 검증 한계의 처리 여부를 확인한다.
현재 main, 원격 브랜치, 기존 태그는 변경하지 않았다.
