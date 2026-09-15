# release #

## v1.3.7 ##
- `DeploymentManagement` gRPC 서비스 추가 (`Open`, `Heartbeat`, `Detach`)
  - 배포를 시작한 CLI의 관리 세션을 조회용 연결과 구분
  - `ManagementCredential`로 재접속 시 동일 관리 세션을 식별
  - `opm/lifecycle`: heartbeat 주기 10초, 마지막 수신 후 만료 기준 20초 및 작업 종료 상태 판정 제공
- `PackageDeployment.Cancel` gRPC API 및 `Target.cancel_supported` 추가
  - 미실행 배포를 취소하고 지연 도착한 실행 요청을 차단하기 위한 계약 제공
  - 실행 중인 작업은 현재 상태를 반환하여 완료 결과를 계속 확인
- 배포 상태·복구 정보 확장
  - `CreateRollout.management` 및 `Rollout.management_session_id`로 배포와 관리 세션 연결
  - 종료 사유(`end_reason`), 마지막·다음 확인 시각(`last_checked_at`, `next_check_at`), 새 배포 제한 여부(`blocks_deployment`) 추가
  - `RolloutConflict` gRPC 오류 상세에 기존 배포 ID·그룹·프로세스 제공
- `Rollout.next_target_start_at` 추가
  - 다음 서버 배포 시작 예정 시각(ms)을 전달하여 대상 간 대기 및 CLI 카운트다운 지원
  - 0은 예정된 대상 간 대기가 없음을 의미하며, 결과 재확인 시각(`next_check_at`)과 구분

## v1.3.6 ##
- `DeploymentHistory` gRPC 서비스 추가 (`List`)
  - rohis가 프로세스별 배포 이력을 gRPC로 조회
  - 레코드: 프로세스, 그룹, 배포 시각(ms), 빌드 사용자·시각, git 브랜치·커밋·메시지
    - artifact에 포함된 `deployment.json`에서 추출
  - 프로세스를 지정하지 않으면 등록된 전체 프로세스의 이력을 최신순으로 반환

## v1.3.5 ##
- gRPC 기반 배포·운영 API 추가 (`opm` 패키지)
  - `opm/api`: 서비스 정의 추가
    - capability 조회(`Discovery`), 인증(`Identity`)
    - 아티팩트(`Artifacts`), 점진 배포(`Deployments`, `PackageDeployment`)
    - 패키지 라우팅·목록(`Routing`, `PackageInventory`)
    - 운영 명령(`CronControl`, `ProcessControl`, `ProcessRegistry`)
  - `opm/transport`: HTTP/gRPC 공용 transport. capability 조회로 서버의 신규 API 지원 여부를 판별하고, 지원하지 않는 서버는 legacy HTTP로 fallback할 수 있도록 `ErrLegacy`를 반환
  - `opm/artifact`: 멀티 플랫폼 아티팩트 검증
  - `opm/operations`, `opm/store`: 작업 진행 상황을 journal로 영속화
    - 재접속(watch), request ID 기반 멱등 요청, 중단된 작업 복구 지원
  - 기존 legacy HTTP 동작은 그대로 유지
- `LogLevelControl` gRPC 서비스 추가 (`List`, `Set`)
  - rolog가 프로세스별 로그레벨을 gRPC로 조회·변경
  - 레벨은 `error`, `warn`, `info`, `debug`, `trace`만 허용
  - juno가 요청을 검증한 뒤 package `loglevels` 파일을 갱신하고, 각 프로세스는 1초 주기 확인에서 이를 반영
- 참고: `ipc` 패키지의 `TestCronExecuteOnyJobName`은 fatima 프로세스 환경 없이 실행하면 nil pointer로 실패함 (v1.3.4부터 동일, 이번 변경과 무관)

## v1.3.4 ##
- 컴포넌트 생명주기(Goaway/Shutdown) 그룹 타임아웃 설정화 및 컴포넌트별 override 지원
  - **기본 그룹 타임아웃 4초 → 30초 상향.** 4초를 초과하던 graceful drain(ALB health check 기반 goaway, Kafka rebalance/commit drain, in-flight flush 등)이 런타임 레벨에서 절단되던 문제 해소
  - config 키 `gofatima.component.lifecycle.timeout.seconds`(초 단위 정수)로 기본 그룹 타임아웃 조정 가능. 미설정·0 이하·비정상 값이면 30초 폴백
  - 컴포넌트가 옵셔널 인터페이스 `fatima.FatimaComponentLifecycleTimeout { LifecycleTimeout() time.Duration }`를 구현하면 자신의 타임아웃을 선언 가능. 0 이하 반환 시 config 기본값으로 폴백
  - 그룹은 동시 실행이므로 그룹 effective timeout = `max(각 컴포넌트의 effective timeout)`. goaway는 `FatimaRuntimeGoaway` 타겟 집합 기준으로만 산정(비타겟 컴포넌트의 선언값은 무시)
  - 정상 컴포넌트는 기존과 동일하게 `done` 우선이라 타임아웃과 무관하게 즉시 다음 그룹으로 진행. 타임아웃 발생 시에도 이후 그룹은 정상 진행("proceeding to next group")
  - Goaway/Shutdown 양쪽에 동일 규칙 적용
  - **주의(동작 변화):** 4s→30s 상향으로 hang(버그) 상태 컴포넌트의 종료가 느려질 수 있음(정상 컴포넌트는 영향 없음)
  - **주의(플랫폼 grace period):** 그룹이 4개(`READER/GENERAL/WRITER/PRE_INIT`)이므로 최악 종료 예산은 `Σ(group_effective_timeout)`까지 늘 수 있음. k8s `terminationGracePeriodSeconds`(기본 30s)/juno 종료 유예를 초과하면 SIGKILL로 잘리므로, 플랫폼 유예에 맞춰 config·컴포넌트 선언값을 조정할 것

## v1.3.3 ##
- FatimaComponent 종료(Goaway/Shutdown) 순서화 및 graceful drain 지원
  - 기존: 모든 컴포넌트를 타입 무관하게 동시 처리
  - 변경: Goaway와 Shutdown 모두 데이터 흐름 기반 그룹 순차 처리 `READER → GENERAL → WRITER → PRE_INIT` (Initialize 순서 `PRE_INIT → WRITER → READER → GENERAL`의 단순 역순이 아님)
    - ingress(READER)를 가장 먼저 내려, 종료 중인 파이프라인으로 신규 요청이 유입되는 것을 차단
  - 그룹 내부는 병렬 유지, 그룹별 `componentGroupTimeout`(기본 4초)으로 hang된 컴포넌트가 전체 종료를 막지 않도록 가드
  - 컴포넌트별 `recover`로 한 컴포넌트의 panic이 다른 컴포넌트 처리에 영향을 주지 않도록 격리
  - `Goaway()`는 `FatimaRuntimeGoaway` 구현체 전체를 대상으로 하며(타입으로 참여 제한 없음), 타입은 처리 순서만 결정
  - 주의: 위 순서화가 실제 graceful shutdown으로 이어지려면 READER 컴포넌트의 `Goaway()`/`Shutdown()`이 `신규 차단 → in-flight 완료 대기 → 반환` 계약을 구현해야 함

## v1.3.2 ##
- config override 로깅 정리 및 base/profile flatten 구조 개선
  - **같은 파일/문서 내부 키 충돌** → `WARN` 유지
    - YAML: dot-notation 키와 nested 맵이 같은 평탄화 키를 생성하는 경우
    - properties: 동일 파일 내 중복 키(다른 값)
  - **base → profile override** → `INFO`로 격하 (이전: `WARN`)
    - multi-doc YAML 내 base 문서 → profile 문서 머지
    - 별도 파일 방식(`application.yaml` → `application.{profile}.yaml/.properties`) 포함
    - 값이 실제로 달라질 때만 로깅. 동일 값으로 덮어쓰는 경우는 침묵
  - 내부 구현: 각 YAML 문서를 독립된 map으로 평탄화(`flattenDoc`)한 뒤 별도 `mergeFlattened` 단계에서 병합. 충돌 판단 위치와 override 판단 위치가 분리되어 로그 의미가 명확해짐
  - `application_loader.go`의 base/profile 파일 간 머지도 동일한 `mergeFlattened` 경로로 통일 (yaml, yml, properties 전 포맷 동일하게 적용)

## v1.3.1 ##
- [yaml 파일에 fatima.profile 추가](https://github.com/fatima-go/fatima-core/issues/25)
  - 단일 `application.yaml` 파일 안에 `---` 구분자로 환경별 설정 블록을 작성 가능 (multi-document YAML)
  - `fatima.profile` 키가 없는 문서를 base(공통값)로 사용하며, `FATIMA_PROFILE` 환경변수 값과 일치하는 블록을 base 위에 머지
  - multi-doc 파일이 감지되면 기존 별도 파일(`application.{profile}.yaml`) 방식은 자동으로 무시됨 (multi-doc 우선, 별도파일 fallback)
  - 매칭되는 profile 블록이 없는 경우 base만 적용 (에러 없음)
  - `FATIMA_PROFILE` 미설정 시 base 문서만 사용
  - `fatima.profile` 키는 결과 Config에서 제거됨 (메타 정보로만 사용)
  - `.yml` 확장자도 동일하게 지원. `properties` 포맷은 single-document 방식 유지
  - `FATIMA_PROFILE` 환경변수를 단일 진입점에서 한 번만 읽어 multi-doc 키 매칭과 별도파일명 결정에 동일하게 적용 (단일 출처 원칙)

## v1.3.0 ##
- [yaml 파일 config 추가 지원](https://github.com/fatima-go/fatima-core/issues/24)
  - **[Breaking]** `{programName}.properties` / `{programName}.{profile}.properties` 파일 검색 제거. `application.*` 이름으로 고정
  - application.properties (또는 yaml/yml)를 base로 로드한 후 application.{profile}.* 파일로 overriding 적용
  - yaml, yml, properties 형식 지원 (우선순위: yaml > yml > properties)
  - yaml nested 키는 점(`.`)으로 평탄화하여 기존 `Config.GetValue("a.b.c")` API와 호환
  - yaml scalar 배열은 쉼표(`,`)로 join하여 단일 string으로 저장. **주의**: 요소 값에 쉼표가 포함되면 분해 시 손상됨
  - yaml 복합 타입 배열(map/nested list)은 미지원 (경고 로그 출력 후 해당 키 스킵)
  - base와 profile override는 동일 형식 페어만 적용 (yaml base + properties override 조합 불가)

## v1.2.2 ##
- [목업 인터페이스 제공](https://github.com/fatima-go/fatima-core/issues/23)


## v1.2.1 ##
- [[bug] ipc session expire 처리 문제 수정 #21](https://github.com/fatima-go/fatima-core/issues/21)

## v1.2.0 ##
- 전체적으로 잘못된 메소드명 수정 : regist -> register<br>
- [보다 나은 IPC 인터페이스 제공](https://github.com/fatima-go/fatima-core/issues/19)

## v1.1.5 ##
- LICENSE.md 추가

## v1.1.4 ##
- [init() 함수내에서 앱 폴더 체크 후 로직 추가](https://github.com/fatima-go/fatima-core/issues/16)

## v1.1.3 ##
- [프로세스 설정에 신규 필드(weight, startsec) 추가 제공](https://github.com/fatima-go/fatima-core/issues/13) : fatima 프로세스 config item 에 weight 항목 추가

## v1.1.2 ##
- [secret init() 함수에서 bouds out of range 에러 수정](https://github.com/fatima-go/fatima-core/issues/8)

## v1.1.1 ##
- [외부에 encrypt 함수를 제공](https://github.com/fatima-go/fatima-core/issues/6)

## v1.1.0 ##
- [alarm 메시지에 부가정보 추가 #5](https://github.com/fatima-go/fatima-core/issues/5)
- [config 처리시 encrypt 기능 제공 #1](https://github.com/fatima-go/fatima-core/issues/1)

## v1.0.0 ##
initial