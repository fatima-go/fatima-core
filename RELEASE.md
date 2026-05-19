# release #

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