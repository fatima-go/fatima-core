# fatima-core #
fatima-core 는 golang을 이용한 프로세스 개발을 돕기 위한 프레임워크이다<br>

## module versioning ##
golang 의 모듈 버전에 대한 정보는 아래 2개의 링크를 참고한다
- [Master Go Module Pseudoversions With GoCenter](https://jfrog.com/blog/go-big-with-pseudo-versions-and-gocenter/)
- [Module version numbering](https://go.dev/doc/modules/version-numbers)

먼저 (현재 개발중인) 소스의 태깅을 위해 현재 브랜치의 가장 최근 커밋 정보를 적절한 포맷으로 출력할 수 있도록 변수를 정의한다
```shell
TZ=UTC git --no-pager show \
   --quiet \
   --abbrev=12 \
   --date='format-local:%Y%m%d%H%M%S' \
   --format="%cd-%h"
```
다음으로 모듈의 pseudo 버저닝을 위해 적절한 버전과 커밋 정보를 조합하여 태깅하고 push를 한다.
```shell
% TZ=UTC git --no-pager show \
   --quiet \
   --abbrev=12 \
   --date='format-local:%Y%m%d%H%M%S' \
   --format="%cd-%h"
20231004081555-6f0f6cc31723
% git tag v0.0.1-20231004081555-6f0f6cc31723
% git push origin v0.0.1-20231004081555-6f0f6cc31723
Total 0 (delta 0), reused 0 (delta 0), pack-reused 0
To https://github.com/fatima-go/fatima-core.git
 * [new tag]         v0.0.1-20231004081555-6f0f6cc31723 -> v0.0.1-20231004081555-6f0f6cc31723
```

# release #
- [release history](./RELEASE.md)

