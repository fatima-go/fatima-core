# YAML 구조화 설정

v1.4.0 / v2.1.0부터 YAML 객체·배열과 값의 타입을 유지한 설정 조회를 지원한다.
기존 `Config` 인터페이스에 필수 메서드를 추가하지 않고, 선택적 `StructuredConfig`를 제공한다.
기존 버전의 태그와 코드는 변경하지 않는다.

## 입력과 병합 결과

`application.yaml`:

```yaml
services:
  group:
    api:
      - name: order-api
        repository: demo/order-api
    backend:
      - name: batch-worker
        repository: demo/batch-worker
```

`application.qa.yaml`:

```yaml
services:
  group:
    api:
      - name: qa-api
        repository: demo/qa-api
```

profile이 `qa`이면 api는 qa-api 한 항목으로 교체되고 backend는 batch-worker를 유지한다.
`services.group.api`라는 설명용 경로는 `services` → `group` → `api`를 뜻한다.
배열 요소를 이름이나 인덱스로 병합하지 않는다.

| override의 내용 | 결과 |
|---|---|
| 키 생략 | base 값 상속 |
| 객체 + 객체 | 내부 키별 재귀 병합 |
| 배열 | override 배열 전체로 교체 |
| `[]` | 존재하는 빈 배열; base 요소 제거 |
| `null`, `~`, 또는 값 없는 `key:` | 존재하는 null; base 값 제거 |
| 객체/배열/scalar 사이의 타입 변경 | override로 교체; 이전 하위 키 제거 |
| `{}` | 객체와 병합하면 기존 내부 키 유지; 없는 키에 지정하면 빈 객체 |

`backend`도 비우려면 qa의 `services.group` 아래에 `backend: []`를 명시한다.
이 규칙은 기존 문자열 getter와 새 구조화 API가 공유하는 하나의 병합 결과에 적용한다.

## 런타임에서 조회

v2 import 경로 예시다. v1에서는 경로의 `/v2`를 제거한다.

```go
import (
    "fmt"
    fatima "github.com/fatima-go/fatima-core/v2"
    "github.com/fatima-go/fatima-core/v2/runtime"
)

func loadAPIs() error {
    config := runtime.GetFatimaRuntime().GetConfig()
    structured, ok := config.(fatima.StructuredConfig)
    if !ok {
        return fmt.Errorf("structured configuration is unavailable")
    }
    var apis []struct {
        Name       string `yaml:"name"`
        Repository string `yaml:"repository"`
    }
    if err := structured.DecodeNode(&apis, "services", "group", "api"); err != nil {
        return err
    }
    // apis에는 qa-api 한 항목이 들어 있다. 업무 의미 검증은 소비 서비스가 담당한다.
    return nil
}
```

`LookupNode(path ...string) (any, bool)`의 각 인자는 **실제 map 키 하나**다.
`LookupNode("a.b")`와 `LookupNode("a", "b")`는 서로 다른 경로다.
인자가 없으면 전체 루트를 반환한다. 빈 문자열 키는 `LookupNode("")`로 조회한다.
배열 인덱스를 경로로 받지는 않는다. 배열 전체를 조회하거나 slice로 디코딩한다.

| 상태 | LookupNode 반환 |
|---|---|
| 경로 없음 / 중간 노드가 객체가 아님 | `nil, false` |
| 명시적 null | `nil, true` |
| 빈 배열 | 길이 0의 `[]any`, `true` |
| 빈 객체 | 길이 0의 `map[string]any`, `true` |

반환된 map/slice와 디코딩된 값을 수정해도 다른 조회 결과는 바뀌지 않는다.
`builder.LoadApplicationConfig`의 `Structured` 필드에서도 같은 스냅샷에 접근할 수 있다.
`NewPropertyConfigReader`, 실제 runtime builder, `MockConfig`의 resources 로더,
`MockFatimaRuntime.SetConfig`를 통한 경로에 연결되어 있다.

## 엄격한 타입 디코딩

- `DecodeNode`의 목적지는 nil이 아닌 포인터여야 한다.
- 문자열 `"10"`을 정수 10으로, 불리언을 문자열로 변환하지 않는다.
- 정수 폭과 signed/unsigned 변환은 범위 안에서만 허용한다. 정수와 실수 간 변환은 하지 않는다.
- struct, slice, 고정 길이 배열, 문자열 키 map, 포인터, `any`를 지원한다.
- struct 필드는 `yaml` 태그 또는 필드 이름의 소문자 표기를 사용한다. `yaml:"-"`는 제외한다.
- 미지의 입력 필드는 무시하고 누락된 struct 필드는 zero value로 둔다.
  `omitempty`, `flow` 태그는 허용하지만 디코딩 결과에 영향을 주지 않는다.
  `inline` 태그와 사용자 정의 unmarshal 메서드는 지원하지 않는다. 등록 항목의 필수값 검증은 소비자가 수행한다.
- null은 포인터/map/slice/interface를 nil로 만든다. 숫자·문자열 등 값 타입으로 디코딩하면 타입 오류다.
- 실패하면 목적지의 기존 값은 유지된다. 성공하면 전체 새 값으로 교체한다.
- YAML timestamp는 yaml.v3의 `time.Time` 타입을 유지한다.

오류는 `*fatima.ConfigNodeError`이며 `Path`, `Expected`를 제공한다. 원본 값은 포함하지 않는다.
`errors.Is`로 `ErrConfigNodeNotFound`, `ErrConfigTypeMismatch`, `ErrInvalidConfigTarget`을 구분한다.
오류 경로의 `[0]` 같은 인자는 진단용 배열 인덱스다.

## 기존 getter의 표현과 신규 버전에서 달라지는 동작

기존 `GetValue/GetString/GetInt/GetBool/GetList`는 계속 사용할 수 있다.
외부 Config 구현체와 mock에 신규 메서드 구현을 요구하지 않는다.

- scalar의 문자열 표현과 기존 GetBool 규칙을 유지한다.
- 단순 YAML 목록은 기존처럼 쉼표로 합쳐 문자열로 제공하고 GetList에서 분리·trim·빈 요소 제거한다.
  `values: ["a,b", "c", "", "  d  "]`는 GetList에서 `[a,b,c,d]`가 된다.
  구조화 조회에서는 원래 네 요소와 공백을 유지한다.
- 문자열 scalar `"a,b"`의 YAML GetList 결과는 한 요소다. `.properties`에서는 쉼표로 분리한다.
- **변경:** `[]`는 기존 GetList에서도 빈 slice를 반환한다. 이전에는 무시되어 base 배열이 남을 수 있었다.
- **변경:** null은 문자열 투영에서 제외한다. GetValue는 false, 다른 getter는 없는 키 오류를 반환한다.
  새 구조화 API에서는 null의 존재를 구분할 수 있다. 이전에는 override의 null이 무시되었다.
- **변경:** 타입 교체 시 이전 하위 키와 목록 메타데이터가 남지 않는다.
- 객체/중첩 배열을 GetList로 요청하면 타입 오류를 반환한다. 정상 로딩 중 unsupported WARN은 없다.
  `YamlSkippedKeys`는 호환 필드로 유지되지만 비어 있다. `YamlComplexKeys`는 GetList로 읽을 수 없는 배열을 표시한다.

## 변수와 secret

프로파일 선택·병합 후 문자열 leaf마다 변수 치환 → secret 해석을 한 번 수행한다.
객체 배열과 중첩 배열의 문자열도 처리한다. map 키, 숫자, 불리언은 치환하지 않는다.
치환 결과가 `"123"`, `"false"`, `"[a,b]"`여도 문자열로 남는다.

논리 키 경로가 `.secret`으로 끝나는 문자열에 기존 `crypt.ResolveSecret`을 적용한다.
배열 인덱스는 논리 경로에 추가하지 않는다. 예를 들어 다음 값은 모두 처리한다.

```yaml
connectors:
  - auth:
      token.secret: "${var.fixture.token}"
secrets:
  secret: [b64:YQ==, b64:Yg==]
```

위 secrets.secret는 구조화 조회에서 `["a", "b"]`, 기존 GetList에서도 `["a", "b"]`다.
이제 YAML secret 배열도 합쳐진 문자열이 아닌 **각 문자열 요소별로** 해석한다.
미해결 변수는 그대로 남는다. secret 해석기의 기존 반환/실패 정책은 유지한다.
기본 해석기의 진단 로그에서 암호문·키·평문을 제거했다. 사용자 정의 해석기의 로그는 그 구현이 책임진다.

## 파일 선택, 충돌과 로딩 실패

기존 선택 규칙을 유지한다.

1. base 형식 우선순위는 yaml → yml → properties다. base가 없으면 profile 전용 파일에서 같은 순서로 선택한다.
2. 단일 YAML 문서는 같은 형식의 별도 profile 파일을 적용한다.
3. 다중 문서는 마지막 base와 일치하는 profile 문서들을 등장 순서대로 병합한다. 별도 profile 파일은 무시한다.
4. 중복 base/profile의 기존 경고는 유지한다. 다중 문서 선택용 fatima.profile은 제거하고 단일 문서의 키는 유지한다.

YAML의 map 키는 문자열이어야 하며, 중복 YAML 키와 잘못된 YAML은 파일 로딩 오류다.
문법 오류 메시지는 입력 값을 노출하지 않도록 정제한다.
문자열 투영에서 점 표기 키와 중첩 키가 충돌하면 경고를 남긴다.
각 객체의 키를 문자열 오름차순으로 순회하고 나중 투영을 사용한다.
따라서 루트의 literal `a.b`가 nested `a → b`보다 우선한다. 서로 다른 문서에서 들어온 키에도 적용한다.
구조화 조회는 두 경로를 모두 보존한다. 혼동을 피하려면 충돌하는 설정을 함께 쓰지 않는다.

`LoadApplicationConfig`의 기존 시그니처와 **파일 로딩 실패 시 WARN 후 계속 진행하는 정책**을 유지한다.
이 API를 엄격한 기동 검증으로 취급하지 않는다. 실패한 파일은 적용하지 않으며,
소비자는 필요한 노드의 존재·타입과 업무 의미를 확인해야 한다. 이번 변경에 엄격한 로딩 진입점이나 동적 재로딩은 포함하지 않는다.

`.properties`의 구조화 조회는 평평한 문자열 키/값만 제공한다.
예를 들어 `a.b=123`은 `LookupNode("a.b")`로 문자열 `"123"`을 얻으며, 중첩 객체나 정수로 추정하지 않는다.

## 서비스 이관

be-knowledge-svc의 기존 별도 파서는 services.group 전체를 나중 문서로 교체한다.
일반 재귀 병합으로 바꾸면 base의 backend가 유지되어 등록 대상이 달라질 수 있다.
의존 버전만 바꾼 뒤 별도 파서를 즉시 제거하지 않는다. 등록 대상·참조·제외·ID를 비교하고,
의도한 결과에 맞춰 profile 설정을 조정한 후 이관한다. 이번 core 변경은 서비스 코드나 등록 정책을 변경하지 않는다.
