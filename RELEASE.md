# release #

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